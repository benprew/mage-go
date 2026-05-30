package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// PreventAttachedFromActivatingNonManaAbilities blocks the attached permanent's
// non-mana activated abilities. Lawmage's Binding pattern.
func TestPreventAttachedFromActivatingNonManaAbilities(t *testing.T) {
	const auraName = "CANA Lawmage-like"
	const creatureName = "CANA Pinger"
	if !mage.CardRegistered(auraName) {
		mage.Register(auraName, func() mage.Card {
			return mage.NewAura(auraName, "{1}{W}{U}",
				mage.WithStaticAbility(
					mage.PreventAttachedFromActivatingNonManaAbilities(core.AttachAura),
				),
			)
		})
	}
	if !mage.CardRegistered(creatureName) {
		mage.Register(creatureName, func() mage.Card {
			return mage.NewCreature(creatureName, "{2}", 1, 1,
				mage.WithActivatedAbility(
					mage.DealDamage(mage.Fixed(1)),
					mage.ManaCostOf("{0}"),
					mage.WithCost(mage.Tap()),
					mage.WithTarget(mage.TargetDamageAnyTarget()),
				),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Plains", 2)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Island", 1)
	tg.AddCard(core.ZoneBattlefield, PlayerA, creatureName)
	tg.AddCard(core.ZoneHand, PlayerA, auraName)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, auraName, creatureName)
	// Try to activate the now-restricted pinger — it should be blocked.
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, creatureName, "PlayerB")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Activation was blocked: PlayerB still at full life. The creature is
	// untapped because the activation was prevented before tapping.
	tg.AssertLife(PlayerB, 20)
	tg.AssertTapped(PlayerA, creatureName, false)
}
