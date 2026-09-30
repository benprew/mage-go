package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestXDamageTargetsIntegration(t *testing.T) {
	const spell = "Synthetic X Damage Recipients"
	const victim = "Synthetic X Damage Victim"
	if !mage.CardRegistered(spell) {
		dsl.Register(spell, func() dsl.Card {
			targets := dsl.TargetDamageRecipientsExactlyX("recipients")
			return dsl.NewSorcery(spell, "{X}", dsl.Spell(dsl.Sequence(dsl.DealDamageEach(dsl.This, targets, dsl.Fixed(2)), dsl.GainLife(dsl.You, dsl.Fixed(1))), dsl.WithTargets(targets)))
		})
		dsl.Register(victim, func() dsl.Card { return dsl.NewCreature(victim, "", 2, 2) })
	}
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, victim)
	tg.AddCard(core.ZoneHand, PlayerA, spell)
	tg.CastSpellWithX(1, core.PrecombatMain, PlayerA, spell, 2, victim, "PlayerB")
	tg.StopAt(1, core.EndCombat)
	tg.Execute()
	tg.AssertPermanentCount(PlayerB, victim, 0)
	tg.AssertGraveyardCount(PlayerB, victim, 1)
	tg.AssertLife(PlayerB, 18)
	tg.AssertLife(PlayerA, 21)
}
