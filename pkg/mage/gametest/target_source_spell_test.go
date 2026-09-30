package gametest

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestPoppedSpellTargetLegalityIntegration(t *testing.T) {
	tg := NewTestGame(t)
	a, b := tg.GetPlayer(PlayerA), tg.GetPlayer(PlayerB)
	recipient := dsl.TargetDamageRecipient("recipient")
	card := dsl.NewInstant("Synthetic Spell Source", "{R}",
		dsl.Spell(dsl.Sequence(dsl.DealDamageFrom(dsl.This, recipient, dsl.Fixed(2)),
			dsl.GainLife(dsl.You, dsl.Fixed(1))), dsl.WithTargets(recipient)))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	victim := tg.PutOnBattlefield(dsl.NewCreature("Synthetic Spell Recipient", "", 2, 9), b.PlayerID())
	if err := tg.CastSpellByName(a.PlayerID(), card.Name(), []uuid.UUID{victim.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	victim.RuntimeAbilities = append(victim.RuntimeAbilities, dsl.ProtectionFromColor(core.Red))
	tg.ResolveTopOfStack()
	tg.AssertLife(PlayerA, 20)
	tg.AssertLife(PlayerB, 20)
	tg.AssertGraveyardCount(PlayerA, card.Name(), 1)
	if tg.Game.FindPermanent(victim.ID()).Damage != 0 {
		t.Fatal("illegal recipient received damage")
	}
}
