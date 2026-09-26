package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestComposableSourceCombatEventsInCombat(t *testing.T) {
	const attackerName = "DSL Attack Watcher"
	const blockerName = "DSL Block Watcher"
	const secondAttackerName = "DSL Plain Attacker"
	if !mage.CardRegistered(attackerName) {
		mage.Register(attackerName, func() mage.Card {
			return mage.NewCreature(attackerName, "", 0, 5, mage.WithAbility(mage.WhenEvent(
				mage.Attacks(mage.This), mage.GainLifeFor(mage.You, mage.Fixed(2)))))
		})
	}
	if !mage.CardRegistered(blockerName) {
		mage.Register(blockerName, func() mage.Card {
			return mage.NewCreature(blockerName, "", 0, 5, mage.WithKeyword(core.CanBlockAny), mage.WithAbility(mage.WhenEvent(
				mage.Blocks(mage.This), mage.GainLifeFor(mage.You, mage.Fixed(3)))))
		})
	}
	if !mage.CardRegistered(secondAttackerName) {
		mage.Register(secondAttackerName, func() mage.Card {
			return mage.NewCreature(secondAttackerName, "", 0, 5)
		})
	}
	for _, tc := range []struct {
		name         string
		block, multi bool
		lifeA, lifeB int
	}{
		{"unblocked", false, false, 22, 20},
		{"blocked", true, false, 22, 23},
		{"blocks two attackers", true, true, 22, 23},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := NewTestGame(t)
			g.AddCard(core.ZoneBattlefield, PlayerA, attackerName)
			g.AddCard(core.ZoneBattlefield, PlayerB, blockerName)
			if tc.multi {
				g.AddCard(core.ZoneBattlefield, PlayerA, secondAttackerName)
				g.Attack(1, PlayerA, attackerName, secondAttackerName)
			} else {
				g.Attack(1, PlayerA, attackerName)
			}
			if tc.block {
				g.Block(1, PlayerB, blockerName, attackerName)
				if tc.multi {
					g.Block(1, PlayerB, blockerName, secondAttackerName)
				}
			}
			g.StopAt(1, core.PostcombatMain)
			g.Execute()
			g.AssertLife(PlayerA, tc.lifeA)
			g.AssertLife(PlayerB, tc.lifeB)
		})
	}
}
