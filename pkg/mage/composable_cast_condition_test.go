package mage

import (
	"strings"
	"testing"
)

func TestControlledCreatureSubtypeAtCastSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name        string
		before      bool
		noncreature bool
		phaseBefore bool
		after       bool
		want        int
	}{
		{name: "controlled at cast, gone at resolution", before: true, want: 23},
		{name: "absent at cast, present at resolution", after: true, want: 20},
		{name: "phased at cast", before: true, phaseBefore: true, want: 20},
		{name: "noncreature subtype at cast", before: true, noncreature: true, want: 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			spell := NewInstant("Test roar", "", Spell(IfCast(ControlledCreatureSubtypeAtCast("Dragon"), GainLifeFor(You, Fixed(3)))))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			var dragon *BaseCard
			if tc.before {
				if tc.noncreature {
					dragon = NewEnchantment("Dragon enchantment", "", WithSubTypes("Dragon"))
				} else {
					dragon = NewCreature("Dragon", "", 2, 2, WithSubTypes("Dragon"))
				}
				g.PutOnBattlefield(dragon, a.PlayerID())
				if tc.phaseBefore {
					g.PhaseOut(dragon.ID())
				}
			}
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
				t.Fatal(err)
			}
			if tc.before && !tc.phaseBefore {
				g.RemoveFromBattlefield(g.FindPermanent(dragon.ID()))
			}
			if tc.after {
				g.PutOnBattlefield(NewCreature("Late dragon", "", 2, 2, WithSubTypes("Dragon")), a.PlayerID())
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if a.Life() != tc.want || clone.GetPlayer(a.PlayerID()).Life() != tc.want {
				t.Fatalf("life=%d clone=%d want=%d", a.Life(), clone.GetPlayer(a.PlayerID()).Life(), tc.want)
			}
		})
	}
}

func TestControlledCreatureSubtypeAtCastDeclaration(t *testing.T) {
	condition := ControlledCreatureSubtypeAtCast("Dragon")
	if got := InspectDefinition(condition); got.Role != "cast-condition" || !strings.Contains(got.Text, "Dragon") {
		t.Fatalf("inspection=%+v", got)
	}
	for _, subtype := range []string{"", "Aura"} {
		card := NewInstant("Invalid", "", Spell(IfCast(ControlledCreatureSubtypeAtCast(subtype), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted subtype %q", subtype)
		}
	}
	valid := NewInstant("Valid", "", Spell(IfCast(condition, DealDamageFrom(This, You, Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	invalidContext := NewArtifact("Invalid context", "", WithAbility(Activate(Costs(), IfCast(condition, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(invalidContext.Name(), invalidContext); err == nil || !strings.Contains(err.Error(), "spell ability") {
		t.Fatalf("activation validation=%v", err)
	}
}

func TestControlledCreatureSubtypeAtCastCopyAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	dragon := NewCreature("Dragon", "", 2, 2, WithSubTypes("Dragon"))
	g.PutOnBattlefield(dragon, a.PlayerID())
	spell := NewInstant("Snapshot", "", Spell(IfCast(ControlledCreatureSubtypeAtCast("Dragon"), GainLifeFor(You, Fixed(2)))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(dragon.ID())
	original := g.StackObjects()[0]
	spellCopy := g.CopyStackObjectDirect(original, b.PlayerID(), false)
	clone := g.Clone()
	spellCopy.CastContext.ControllerCreatureSubtypesAtCast["Dragon"] = false
	if !original.CastContext.ControllerCreatureSubtypesAtCast["Dragon"] {
		t.Fatal("copy changed original snapshot")
	}
	if !clone.StackObjects()[0].CastContext.ControllerCreatureSubtypesAtCast["Dragon"] {
		t.Fatal("copy changed clone snapshot")
	}
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone resolution changed original")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 20 {
		t.Fatalf("life A=%d B=%d", a.Life(), b.Life())
	}
}
