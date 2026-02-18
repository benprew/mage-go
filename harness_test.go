package mage

import "testing"

func TestHarnessSmokeTest(t *testing.T) {
	if !CardRegistered("Smoke Test Creature") {
		Register("Smoke Test Creature", func() Card {
			c := NewCreature("Smoke Test Creature", "{1}{G}", "Beast")
			c.power = 2
			c.toughness = 2
			return c
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(ZoneBattlefield, PlayerA, "Smoke Test Creature")
	tg.StopAt(1, PrecombatMain)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Smoke Test Creature", 1)
	tg.AssertPowerToughness(PlayerA, "Smoke Test Creature", 2, 2)
	tg.AssertLife(PlayerA, 20)
	tg.AssertLife(PlayerB, 20)
}
