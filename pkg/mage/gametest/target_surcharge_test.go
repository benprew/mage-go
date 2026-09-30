package gametest

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTargetSurchargeIntegration(t *testing.T) {
	tg := NewTestGame(t)
	tg.SetStep(core.PrecombatMain)
	tg.SetActivePlayerIndex(0)
	a, b := tg.GetPlayer(PlayerA), tg.GetPlayer(PlayerB)
	recipients := dsl.TargetDamageRecipientsAnyNumber("recipients")
	card := dsl.NewSorcery("Synthetic Target Surcharge", "{X}{R}",
		dsl.Spell(dsl.DealDamageEvenly(dsl.This, recipients, dsl.ValueAtResolution(dsl.XValue())),
			dsl.WithTargets(recipients), dsl.AdditionalManaPerTargetBeyondFirst(1)))
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
		t.Fatal("wrong target surcharge payment")
	}
}
