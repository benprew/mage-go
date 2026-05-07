package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// AssignsDamageEqualToToughnessForMatching makes affected creatures assign
// combat damage equal to toughness rather than power. Assault Formation
// pattern: "Each creature you control assigns combat damage equal to its
// toughness rather than its power."
func TestAssignsDamageEqualToToughnessForMatching(t *testing.T) {
	const enchName = "ADET Assault-like"
	const wallName = "ADET Wall 0/4"
	if !mage.CardRegistered(enchName) {
		mage.Register(enchName, func() mage.Card {
			return mage.NewEnchantment(enchName, "{1}{G}",
				mage.WithStaticAbility(
					mage.AssignsDamageEqualToToughnessForCreaturesYouControl(),
				),
			)
		})
	}
	if !mage.CardRegistered(wallName) {
		mage.Register(wallName, func() mage.Card {
			return mage.NewCreature(wallName, "{G}", 0, 4,
				mage.WithSubTypes("Wall"),
				// Grant attack capability for the test (normally Walls have defender).
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, enchName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, wallName)
	tg.Attack(3, PlayerA, wallName)
	tg.StopAt(3, core.EndStep)
	tg.Execute()

	// 0/4 attacker normally deals 0; with toughness-instead-of-power, deals 4.
	tg.AssertLife(PlayerB, 16)
}
