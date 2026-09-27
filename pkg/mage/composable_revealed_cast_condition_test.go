package mage

import (
	"strings"
	"testing"
)

func TestRevealedSubtypeAtCastPaymentSnapshot(t *testing.T) {
	for _, tc := range []struct {
		name   string
		accept bool
		card   bool
		want   int
	}{
		{"paid Dragon reveal", true, true, 23},
		{"declined reveal", false, true, 20},
		{"no matching card", true, false, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: tc.accept}
			g := NewGame(a, NewBasePlayer("B"))
			paid, revealed := BindNumber("paid"), BindCard("revealed")
			spell := NewInstant("Reveal test", "", WithAbility(Spell(
				IfCast(RevealedSubtypeAtCast("Dragon"), GainLifeFor(You, Fixed(3))))),
				WithAdditionalCost(OptionalRevealFromHandCost(HasSubTypeCardFilter("Dragon"), "Reveal Dragon", "Reveal?", paid, revealed)))
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if tc.card {
				a.AddToHand(NewCreature("Dragon", "", 2, 2, WithSubTypes("Dragon")))
			} else {
				a.AddToHand(NewCreature("Bear", "", 2, 2))
			}
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if a.Life() != tc.want {
				t.Fatalf("life = %d, want %d", a.Life(), tc.want)
			}
		})
	}
}

func TestRevealedSubtypeAtCastImmutableAcrossCopies(t *testing.T) {
	a, b := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}, NewBasePlayer("B")
	g := NewGame(a, b)
	paid, revealed := BindNumber("paid"), BindCard("revealed")
	spell := NewInstant("Reveal test", "", WithAbility(Spell(
		IfCast(RevealedSubtypeAtCast("Dragon"), GainLifeFor(You, Fixed(3))))),
		WithAdditionalCost(OptionalRevealFromHandCost(HasSubTypeCardFilter("Dragon"), "Reveal Dragon", "Reveal?", paid, revealed)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	dragon := NewCreature("Dragon", "", 2, 2, WithSubTypes("Dragon"))
	a.AddToHand(dragon)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	original := g.StackObjects()[0]
	spellCopy := g.CopyStackObjectDirect(original, b.PlayerID(), false)
	clone := g.Clone()
	if _, ok := a.RemoveFromHand(dragon.ID()); !ok {
		t.Fatal("revealed card did not leave hand")
	}
	a.AddToHand(dragon)
	dragon.AddSubType("Elf")
	spellCopy.CastContext.RevealedAtCast[0].AddSubType("Goblin")
	if original.CastContext.RevealedSubtypeAtCast("Elf") || original.CastContext.RevealedSubtypeAtCast("Goblin") || clone.StackObjects()[0].CastContext.RevealedSubtypeAtCast("Goblin") {
		t.Fatal("cast snapshot changed after a card or copy changed")
	}
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone resolution changed live players")
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 23 {
		t.Fatalf("life A=%d B=%d", a.Life(), b.Life())
	}
}

func TestRevealedSubtypeAtCastDeclaration(t *testing.T) {
	condition := RevealedSubtypeAtCast("Dragon")
	if got := InspectDefinition(condition); got.Role != "cast-condition" || !strings.Contains(got.Text, "Dragon") {
		t.Fatalf("inspection = %+v", got)
	}
	invalid := NewInstant("Invalid", "", Spell(IfCast(RevealedSubtypeAtCast(""), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
		t.Fatal("accepted empty subtype")
	}
	valid := NewInstant("Valid", "", Spell(IfCast(RevealedSubtypeAtCast("Aura"), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), IfCast(condition, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("accepted cast condition in activation")
	}
}
