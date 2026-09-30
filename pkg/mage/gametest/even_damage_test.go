package gametest

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestEvenDamageIntegration(t *testing.T) {
	tg := NewTestGame(t)
	a, b := tg.GetPlayer(PlayerA), tg.GetPlayer(PlayerB)
	recipients := dsl.TargetDamageRecipientsAnyNumber("recipients")
	card := dsl.NewInstant("Synthetic Equal Division", "{X}{R}",
		dsl.Spell(dsl.DealDamageEvenly(dsl.This, recipients, dsl.ValueAtResolution(dsl.XValue())), dsl.WithTargets(recipients)))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	a.ManaPool().Add(core.Colorless, 5)
	victim := tg.PutOnBattlefield(dsl.NewCreature("Synthetic Recipient", "", 1, 9), b.PlayerID())
	if err := tg.CastSpellByName(a.PlayerID(), card.Name(), []uuid.UUID{victim.ID(), b.PlayerID()}, 5); err != nil {
		t.Fatal(err)
	}
	victim.RuntimeAbilities = append(victim.RuntimeAbilities, dsl.ProtectionFromColor(core.Red))
	tg.ResolveTopOfStack()
	tg.AssertLife(PlayerA, 20)
	tg.AssertLife(PlayerB, 15)
	tg.AssertGraveyardCount(PlayerA, card.Name(), 1)
	if victim.Damage != 0 {
		t.Fatal("illegal recipient received damage")
	}
}
