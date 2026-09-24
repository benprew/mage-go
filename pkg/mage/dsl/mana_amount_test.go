package dsl_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedManaAmountComposition(t *testing.T) {
	target := dsl.TargetPermanent("amount", dsl.IsCreature)
	for _, ability := range []mage.Ability{
		dsl.Spell(dsl.AddManaAmount(core.Black, dsl.PowerOf(target)), dsl.WithTargets(target)),
		dsl.SpellModes(dsl.ModeOf("Mana", dsl.AddManaAmount(core.Black, dsl.Fixed(2))), dsl.ModeOf("Life", dsl.GainLife(dsl.You, dsl.Fixed(2)))),
		dsl.At(dsl.YourFirstMainPhase, dsl.AddManaAmount(core.Black, dsl.Fixed(2))),
		dsl.Activate(dsl.Costs(dsl.TapCost(dsl.This)), dsl.AddManaAmount(core.Black, dsl.Fixed(2))),
	} {
		card := dsl.NewArtifact("Mana", "", dsl.WithAbility(ability))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	source := dsl.NewCreature("Amount", "", 3, 4)
	g.PutOnBattlefield(source, a.PlayerID())
	ability := dsl.Activate(dsl.Costs(), dsl.Sequence(dsl.AddManaAmount(core.Black, dsl.PowerOf(target)), dsl.GainLife(dsl.You, dsl.Fixed(1))), dsl.WithTargets(target))
	card := dsl.NewArtifact("Targeted mana", "", dsl.WithAbility(ability))
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{source.ID()}); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 1 || a.ManaPool().Count(core.Black) != 0 {
		t.Fatal("targeted mana bypassed stack")
	}
	g.ResolveStack()
	if a.ManaPool().Count(core.Black) != 3 || a.Life() != 21 {
		t.Fatal("targeted amount lost resolution context")
	}

}
