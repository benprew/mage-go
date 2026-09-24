package dsl_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestTypedLibraryChoiceResolution(t *testing.T) {
	for _, surveil := range []bool{false, true} {
		for _, amount := range []int{-1, 0, 2, 5} {
			a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
			g := mage.NewGame(a, b)
			top, middle, bottom := mage.NewLand("Top"), mage.NewLand("Middle"), mage.NewLand("Bottom")
			b.SetLibrary([]mage.Card{top, middle, bottom})
			source := mage.NewCreature("Amount", "", amount, 6)
			g.PutOnBattlefield(source, a.PlayerID())
			saved := dsl.BindPermanent("saved")
			value := dsl.PowerOf(dsl.From(saved))
			choice := dsl.Scry(value)
			if surveil {
				choice = dsl.Surveil(value)
			}
			body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), choice)
			if amount > 0 {
				if surveil {
					b.AddSurveilDecision([]string{"Top"}, nil)
				} else {
					b.AddScryDecision([]string{"Top"}, nil)
				}
			}
			if err := mage.ApplyEffect(g, body, source.ID(), b.PlayerID(), []uuid.UUID{a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if amount <= 0 {
				if len(b.Library()) != 3 || b.Library()[0].ID() != top.ID() || len(b.Graveyard()) != 0 {
					t.Fatal("nonpositive amount changed zones")
				}
			} else if surveil {
				if len(b.Library()) != 2 || b.Library()[0].ID() != middle.ID() || len(b.Graveyard()) != 1 || b.Graveyard()[0].ID() != top.ID() {
					t.Fatal("surveil lost binding, recipient, or placement")
				}
			} else if len(b.Library()) != 3 || b.Library()[0].ID() != middle.ID() || b.Library()[2].ID() != top.ID() {
				t.Fatal("scry lost binding, recipient, or placement")
			}
			if len(a.Library()) != 0 || len(a.Graveyard()) != 0 {
				t.Fatal("changed source controller zones")
			}
		}
	}
}

func TestTypedLibraryChoiceComposition(t *testing.T) {
	for _, build := range []func(dsl.ResolutionValue) dsl.ResolvingEffect{dsl.Scry, dsl.Surveil} {
		target := dsl.TargetPermanent("creature", dsl.IsCreature)
		body := build(dsl.MaxValue(dsl.Fixed(1), dsl.PowerOf(target)))
		for _, card := range []mage.Card{
			dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(body, dsl.WithTargets(target)))),
			dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Choice", body, dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
			dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body, dsl.WithTargets(target)))),
			dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body, dsl.WithTargets(target)))),
		} {
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
		}
	}
}
