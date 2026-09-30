package gametest

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestDepartedSourceTargetLegalityIntegration(t *testing.T) {
	tg := NewTestGame(t)
	a, b := tg.GetPlayer(PlayerA), tg.GetPlayer(PlayerB)
	recipient := dsl.TargetDamageRecipient("recipient")
	card := dsl.NewCreature("Synthetic Departed Source", "{G}", 2, 2,
		dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.SacrificeSourceCost()),
			dsl.Sequence(dsl.DealDamageFrom(dsl.This, recipient, dsl.Fixed(2)),
				dsl.GainLife(dsl.You, dsl.Fixed(1))), dsl.WithTargets(recipient))))
	source := tg.PutOnBattlefield(card, a.PlayerID())
	victim := tg.PutOnBattlefield(dsl.NewCreature("Synthetic Protected Recipient", "", 2, 9), b.PlayerID())
	if err := tg.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	tg.Game.PutOnBattlefield(card, b.PlayerID()).FaceDown = true
	victim.RuntimeAbilities = append(victim.RuntimeAbilities, dsl.ProtectionFromColor(core.Green))
	tg.ResolveStack()
	tg.AssertLife(PlayerA, 20)
	tg.AssertLife(PlayerB, 20)
	if tg.Game.FindPermanent(victim.ID()).Damage != 0 {
		t.Fatal("illegal recipient received damage")
	}
}
