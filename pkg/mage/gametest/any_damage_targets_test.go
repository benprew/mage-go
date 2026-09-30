package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestAnyNumberDamageTargetsIntegration(t *testing.T) {
	const spell = "Synthetic Any Number Damage"
	const first = "Synthetic Any Number Recipient One"
	const second = "Synthetic Any Number Recipient Two"
	if !mage.CardRegistered(spell) {
		dsl.Register(spell, func() dsl.Card {
			recipients := dsl.TargetDamageRecipientsAnyNumber("recipients")
			return dsl.NewSorcery(spell, "{X}", dsl.Spell(dsl.Sequence(dsl.DealAllocatedDamageX(dsl.This, recipients), dsl.GainLife(dsl.You, dsl.Fixed(1))), dsl.WithTargets(recipients)))
		})
		dsl.Register(first, func() dsl.Card { return dsl.NewCreature(first, "", 2, 2) })
		dsl.Register(second, func() dsl.Card { return dsl.NewCreature(second, "", 2, 2) })
	}
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, first)
	tg.AddCard(core.ZoneBattlefield, PlayerB, second)
	tg.AddCard(core.ZoneHand, PlayerA, spell)
	tg.ChooseDamageDistribution(PlayerA, map[string]int{first: 1, second: 2})
	tg.CastSpellWithX(1, core.PrecombatMain, PlayerA, spell, 3, first, second)
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	tg.AssertPermanentCount(PlayerB, first, 1)
	tg.AssertPermanentCount(PlayerB, second, 0)
	tg.AssertGraveyardCount(PlayerB, second, 1)
	tg.AssertGraveyardCount(PlayerA, spell, 1)
	tg.AssertLife(PlayerA, 21)
}
