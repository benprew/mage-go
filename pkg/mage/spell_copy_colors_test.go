package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func applyColorCopy(t *testing.T, g *Game, original *StackObject, controller uuid.UUID, colors ...core.Color) *StackObject {
	t.Helper()
	binding := BindSpell("original")
	g.resolution.spellBindings = map[*SpellBinding]uuid.UUID{binding: original.ID}
	if err := CopyInstantOrSorceryWithColors(FromSpell(binding), colors...).Apply(&EffectContext{Game: g, Controller: controller}); err != nil {
		t.Fatal(err)
	}
	return g.StackPeek()
}

func TestSpellCopyColorsIdentity(t *testing.T) {
	g, a, b := newSpellCopyGame()
	card := NewInstant("Blue damage", "{U}", NewTargetedSpell(TargetDamageAnyTarget(), DealDamage(Fixed(3))))
	original := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID(), Effects: []Effect{DealDamage(Fixed(3))}, Targets: []uuid.UUID{b.PlayerID()}, XValue: 7, ModeChoice: 2, DamageDistribution: map[uuid.UUID]int{b.PlayerID(): 3}}
	g.pushStack(original)
	g.ChangeObjectColor(original.ID, core.Green)
	b.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
	colors := []core.Color{core.Red}
	cp := applyColorCopy(t, g, original, b.PlayerID(), colors...)
	colors[0] = core.Black
	if cp == original || !cp.IsCopy || cp.SourceID == original.SourceID || cp.Card.Owner() != b.PlayerID() || cp.Controller != b.PlayerID() || cp.XValue != 7 || cp.ModeChoice != 2 || cp.Targets[0] != a.PlayerID() || cp.DamageDistribution[a.PlayerID()] != 3 || !slices.Equal(g.EffectiveColors(cp.ID), []core.Color{core.Red}) || !slices.Equal(g.EffectiveColors(original.ID), []core.Color{core.Green}) {
		t.Fatal("copy identity, decisions or colors")
	}
	clone := g.Clone()
	g.ChangeObjectColor(cp.ID, core.White)
	next := g.CopyStackObjectDirect(cp, a.PlayerID(), false)
	if !slices.Equal(g.EffectiveColors(next.ID), []core.Color{core.Red}) || !slices.Equal(clone.EffectiveColors(cp.ID), []core.Color{core.Red}) {
		t.Fatal("ordinary color or clone changed copiable colors")
	}
	g.ResolveStack()
	clone.ResolveStack()
	if a.Life() != 14 || b.Life() != 17 || clone.GetPlayer(a.PlayerID()).Life() != 17 || clone.GetPlayer(b.PlayerID()).Life() != 17 || len(b.Graveyard()) != 0 {
		t.Fatal("copy gameplay or zone lifetime")
	}
}

func TestSpellCopyColorsRetarget(t *testing.T) {
	for _, choice := range []string{"legal", "protected", "cancel", "phase", "return"} {
		t.Run(choice, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			card := NewInstant("Blue damage", "{U}", NewTargetedSpell(TargetCreature(), DealDamage(Fixed(3))))
			old := g.PutOnBattlefield(NewCreature("Old", "", 1, 8), a.PlayerID())
			protected := g.PutOnBattlefield(NewCreature("Protected", "", 1, 8, WithAbility(ProtectionFromColors(core.Red))), a.PlayerID())
			legal := g.PutOnBattlefield(NewCreature("Legal", "", 1, 8), b.PlayerID())
			original := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID(), Effects: []Effect{DealDamage(Fixed(3))}, Targets: []uuid.UUID{old.ID()}, targetIncarnations: map[uuid.UUID]uuid.UUID{old.ID(): old.incarnationID}}
			g.pushStack(original)
			switch choice {
			case "legal":
				b.chooseQueue = [][]uuid.UUID{{legal.ID()}}
			case "protected":
				b.chooseQueue = [][]uuid.UUID{{protected.ID()}}
			case "cancel":
				b.chooseQueue = [][]uuid.UUID{nil}
			case "phase":
				g.PhaseOut(old.ID())
				b.chooseQueue = [][]uuid.UUID{nil}
			case "return":
				g.RemoveFromBattlefield(old)
				g.PutOnBattlefield(old.Card, a.PlayerID())
				b.chooseQueue = [][]uuid.UUID{nil}
			}
			cp := applyColorCopy(t, g, original, b.PlayerID(), core.Red)
			want := old.ID()
			if choice == "legal" {
				want = legal.ID()
			}
			if cp.Targets[0] != want || original.Targets[0] != old.ID() {
				t.Fatal("illegal retarget or changed original")
			}
			g.ResolveTopOfStack()
			if g.FindPermanent(protected.ID()).Damage != 0 {
				t.Fatal("red protection did not apply before choice")
			}
			if choice == "legal" && g.FindPermanent(legal.ID()).Damage != 3 {
				t.Fatal("legal new target")
			}
			if (choice == "protected" || choice == "cancel") && g.FindPermanent(old.ID()).Damage != 3 {
				t.Fatal("retained target")
			}
			if choice == "return" && g.FindPermanent(old.ID()).Damage != 0 {
				t.Fatal("returned target replaced saved visit")
			}
		})
	}
}

func TestSpellCopyColorsDefinition(t *testing.T) {
	target := DeclareSpellTarget("target")
	for _, colors := range [][]core.Color{{core.Red}, {core.Blue, core.Green}, {}} {
		card := NewInstant("Copy", "", WithAbility(Spell(CopyInstantOrSorceryWithColors(target, colors...), WithTargets(target))))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	for _, body := range []ResolvingEffect{CopyInstantOrSorceryWithColors(nil, core.Red), CopyInstantOrSorceryWithColors(target, core.AnyColor), CopyInstantOrSorceryWithColors(target, core.Colorless)} {
		card := NewInstant("Bad", "", WithAbility(Spell(body, WithTargets(target))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid definition")
		}
	}
	card := NewInstant("Unbound", "", WithAbility(Spell(CopyInstantOrSorceryWithColors(target, core.Red))))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("unbound reference")
	}
	g, a, _ := newSpellCopyGame()
	binding := BindSpell("absent")
	if err := CopyInstantOrSorceryWithColors(FromSpell(binding), core.Red).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil || g.StackSize() != 0 {
		t.Fatal("absent reference used LKI")
	}
}

func TestSpellCopyColorsExceptions(t *testing.T) {
	for _, colors := range [][]core.Color{{}, {core.Red}, {core.Blue, core.Green}} {
		g, a, b := newSpellCopyGame()
		plan, err := NewCardDefinitionPlan(func() Card {
			return NewInstant("Color definition", "{U}", WithStaticAbility(&testCopyColorDefinition{}), WithAbility(Spell(GainLifeFor(You, Fixed(2)))))
		})
		if err != nil {
			t.Fatal(err)
		}
		card, err := plan.Build()
		if err != nil {
			t.Fatal(err)
		}
		original := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID(), Effects: []Effect{GainLifeFor(You, Fixed(2))}}
		g.pushStack(original)
		cp := applyColorCopy(t, g, original, b.PlayerID(), colors...)
		base := cp.Card.(*BaseCard)
		if !slices.Equal(g.EffectiveColors(cp.ID), colors) || len(base.Abilities()) != 1 || len(original.Card.Abilities()) != 2 {
			t.Fatal("color definition omission")
		}
		prototype, err := base.copiableValues.Build()
		if err != nil || !slices.Equal(cardColors(prototype), colors) || len(prototype.Abilities()) != 1 {
			t.Fatal("copy provenance", err)
		}
		g.ResolveStack()
		if a.Life() != 22 || b.Life() != 22 {
			t.Fatal("rules or controller")
		}
	}
}
