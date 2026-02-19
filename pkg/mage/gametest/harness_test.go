package gametest

import (
	"testing"

	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

func TestHarnessSmokeTest(t *testing.T) {
	if !mage.CardRegistered("Smoke Test Creature") {
		mage.Register("Smoke Test Creature", func() mage.Card {
			c := mage.NewCreature("Smoke Test Creature", "{1}{G}", 2, 2, "Beast")
			return c
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Smoke Test Creature")
	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Smoke Test Creature", 1)
	tg.AssertPowerToughness(PlayerA, "Smoke Test Creature", 2, 2)
	tg.AssertLife(PlayerA, 20)
	tg.AssertLife(PlayerB, 20)
}
