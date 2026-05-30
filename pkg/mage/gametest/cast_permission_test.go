package gametest

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// CastAsThoughHadFlash grants the controller permission to cast cards
// matching the filter at instant speed. Used by Rattlechains for Spirit
// spells. Without the grant, casting a sorcery-speed creature during the
// opponent's turn would fail with ErrSorcerySpeed.
func TestCastAsThoughHadFlash_LiftsSorcerySpeedGate(t *testing.T) {
	const cardName = "CASTPERM Spirit Granter"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			isSpirit := mage.NewCardFilter("Spirit creature", func(c mage.Card) bool {
				return c.HasType(core.TypeCreature) && c.HasSubType("Spirit")
			})
			return mage.NewEnchantment(cardName, "{0}",
				mage.WithStaticAbility(mage.CastAsThoughHadFlash(isSpirit)),
			)
		})
	}

	const spiritName = "CASTPERM Spirit Token Card"
	if !mage.CardRegistered(spiritName) {
		mage.Register(spiritName, func() mage.Card {
			return mage.NewCreature(spiritName, "{0}", 1, 1, mage.WithSubTypes("Spirit"))
		})
	}

	// PlayerA owns the granter. PlayerB casts a creature during A's end step.
	// Without the grant, casting the Spirit at instant speed would fail; we
	// assert the spell resolved by checking the permanent count.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, cardName)
	tg.AddCard(core.ZoneHand, PlayerB, spiritName)
	tg.CastSpell(1, core.EndStep, PlayerB, spiritName)
	tg.StopAt(2, core.Untap)
	tg.Execute()

	tg.AssertPermanentCount(PlayerB, spiritName, 1)
}

// CastAsThoughHadFlash does NOT lift the gate for cards outside the filter.
func TestCastAsThoughHadFlash_FilterRestricts(t *testing.T) {
	const cardName = "CASTPERM Spirit-only Granter"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			isSpirit := mage.NewCardFilter("Spirit creature", func(c mage.Card) bool {
				return c.HasType(core.TypeCreature) && c.HasSubType("Spirit")
			})
			return mage.NewEnchantment(cardName, "{0}",
				mage.WithStaticAbility(mage.CastAsThoughHadFlash(isSpirit)),
			)
		})
	}

	const goblinName = "CASTPERM Goblin"
	if !mage.CardRegistered(goblinName) {
		mage.Register(goblinName, func() mage.Card {
			return mage.NewCreature(goblinName, "{0}", 1, 1, mage.WithSubTypes("Goblin"))
		})
	}

	// PlayerB tries to cast a Goblin (not a Spirit) during A's end step.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerB, cardName)
	tg.AddCard(core.ZoneHand, PlayerB, goblinName)
	tg.CastSpell(1, core.EndStep, PlayerB, goblinName)
	tg.StopAt(2, core.Untap)
	tg.Execute()

	// Cast attempt failed — the goblin is still in hand and there is no
	// goblin permanent on the battlefield.
	tg.AssertPermanentCount(PlayerB, goblinName, 0)
}
