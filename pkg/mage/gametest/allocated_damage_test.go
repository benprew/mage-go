package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAllocatedDamageIntegration(t *testing.T) {
	const spell = "Synthetic Allocated Damage"
	const first = "Synthetic Allocation Recipient One"
	const second = "Synthetic Allocation Recipient Two"
	if !mage.CardRegistered(spell) {
		dsl.Register(spell, func() dsl.Card {
			recipients := dsl.TargetDamageRecipients("recipients", 1, 3)
			return dsl.NewSorcery(spell, "", dsl.Spell(dsl.DealAllocatedDamage(dsl.This, recipients, 3), dsl.WithTargets(recipients)))
		})
		dsl.Register(first, func() dsl.Card { return dsl.NewCreature(first, "", 2, 2) })
		dsl.Register(second, func() dsl.Card { return dsl.NewCreature(second, "", 2, 2) })
	}
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, first)
	tg.AddCard(core.ZoneBattlefield, PlayerB, second)
	tg.AddCard(core.ZoneHand, PlayerA, spell)
	tg.ChooseDamageDistribution(PlayerA, map[string]int{first: 1, second: 2})
	tg.CastSpell(1, core.PrecombatMain, PlayerA, spell, first, second)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	tg.AssertPermanentCount(PlayerB, first, 1)
	tg.AssertPermanentCount(PlayerB, second, 0)
	tg.AssertGraveyardCount(PlayerB, second, 1)
	tg.AssertGraveyardCount(PlayerA, spell, 1)
}
