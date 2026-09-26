package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDeclaredBlockersEmitOneBecomesBlockedEvent(t *testing.T) {
	const attackerName = "Test Becomes Blocked Attacker"
	const restrictedName = "Test Becomes Blocked Restricted"
	const blockerAName = "Test Becomes Blocked First"
	const blockerBName = "Test Becomes Blocked Second"
	if !mage.CardRegistered(attackerName) {
		mage.Register(attackerName, func() mage.Card {
			return mage.NewCreature(attackerName, "", 4, 4, mage.WithAbility(
				mage.WhenEvent(mage.BecomesBlocked(mage.This), mage.GainLifeFor(mage.You, mage.Fixed(1)))))
		})
		mage.Register(blockerAName, func() mage.Card { return mage.NewCreature(blockerAName, "", 1, 1) })
		mage.Register(blockerBName, func() mage.Card { return mage.NewCreature(blockerBName, "", 1, 1) })
		mage.Register(restrictedName, func() mage.Card {
			return mage.NewCreature(restrictedName, "", 4, 4,
				mage.WithAbility(mage.WhenEvent(mage.BecomesBlocked(mage.This),
					mage.GainLifeFor(mage.You, mage.Fixed(1)))),
				mage.WithStaticAbility(mage.SourceCantBeBlockedByFewerThan(3)))
		})
	}
	for _, tc := range []struct {
		name string
		life int
	}{{attackerName, 21}, {restrictedName, 20}} {
		t.Run(tc.name, func(t *testing.T) {
			g := NewTestGame(t)
			g.AddCard(core.ZoneBattlefield, PlayerA, tc.name)
			g.AddCard(core.ZoneBattlefield, PlayerB, blockerAName)
			g.AddCard(core.ZoneBattlefield, PlayerB, blockerBName)
			g.Attack(1, PlayerA, tc.name)
			g.Block(1, PlayerB, blockerAName, tc.name)
			g.Block(1, PlayerB, blockerBName, tc.name)
			g.StopAt(1, core.PostcombatMain)
			g.Execute()
			g.AssertLife(PlayerA, tc.life)
		})
	}
}
