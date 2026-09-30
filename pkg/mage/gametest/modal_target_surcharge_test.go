package gametest

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestModalTargetSurchargeIntegration(t *testing.T) {
	tg := NewTestGame(t)
	tg.SetStep(core.PrecombatMain)
	tg.SetActivePlayerIndex(0)
	a, b := tg.GetPlayer(PlayerA), tg.GetPlayer(PlayerB)
	targets := dsl.TargetDamageRecipientsAnyNumber("recipients")
	card := dsl.NewInstant("Synthetic Modal Surcharge", "{X}{R}",
		dsl.WithAbility(dsl.SpellModesWithOptions(
			[]dsl.ModalSpellOption{dsl.AdditionalManaPerTargetBeyondFirst(1)},
			dsl.ModeOf("Damage", dsl.DealDamageEvenly(dsl.This, targets, dsl.ValueAtResolution(dsl.XValue())), dsl.WithTargets(targets)),
			dsl.ModeOf("Life", dsl.GainLife(dsl.You, dsl.Fixed(3))))))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	a.ManaPool().Add(core.Colorless, 5)
	if err := tg.CastSpellByName(a.PlayerID(), card.Name(), []uuid.UUID{a.PlayerID(), b.PlayerID()}, 4); err != nil {
		t.Fatal(err)
	}
	tg.ResolveTopOfStack()
	tg.AssertLife(PlayerA, 18)
	tg.AssertLife(PlayerB, 18)
	tg.AssertGraveyardCount(PlayerA, card.Name(), 1)
	if a.ManaPool().TotalMana() != 0 {
		t.Fatal("wrong modal surcharge payment")
	}
}
