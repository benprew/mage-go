package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDeclaredBlockersEmitPerCreatureBlockedByEvent(t *testing.T) {
	const attackerName = "Test Pair Attacker"
	const restrictedName = "Test Pair Restricted Attacker"
	const firstName = "Test Pair First"
	const secondName = "Test Pair Second"
	if !mage.CardRegistered(attackerName) {
		mage.Register(attackerName, func() mage.Card {
			return mage.NewCreature(attackerName, "", 4, 4, mage.WithAbility(
				mage.NewTriggered(core.EvtCreatureBlockedBy, false,
					mage.GainLifeFor(mage.You, mage.Fixed(1))).SetConditionData(mage.EventSourceIsSelf{})))
		})
		mage.Register(firstName, func() mage.Card { return mage.NewCreature(firstName, "", 1, 1) })
		mage.Register(secondName, func() mage.Card { return mage.NewCreature(secondName, "", 1, 1) })
		mage.Register(restrictedName, func() mage.Card {
			return mage.NewCreature(restrictedName, "", 4, 4,
				mage.WithAbility(mage.NewTriggered(core.EvtCreatureBlockedBy, false,
					mage.GainLifeFor(mage.You, mage.Fixed(1))).SetConditionData(mage.EventSourceIsSelf{})),
				mage.WithStaticAbility(mage.SourceCantBeBlockedByFewerThan(3)))
		})
	}
	for _, tc := range []struct {
		name string
		life int
	}{{attackerName, 22}, {restrictedName, 20}} {
		t.Run(tc.name, func(t *testing.T) {
			g := NewTestGame(t)
			g.AddCard(core.ZoneBattlefield, PlayerA, tc.name)
			g.AddCard(core.ZoneBattlefield, PlayerB, firstName)
			g.AddCard(core.ZoneBattlefield, PlayerB, secondName)
			g.Attack(1, PlayerA, tc.name)
			g.Block(1, PlayerB, firstName, tc.name)
			g.Block(1, PlayerB, secondName, tc.name)
			g.StopAt(1, core.PostcombatMain)
			g.Execute()
			g.AssertLife(PlayerA, tc.life)
		})
	}
}
