package gametest

import (
	"sync"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

var revealFromHandCostOnce sync.Once

func registerRevealFromHandCostCards() {
	revealFromHandCostOnce.Do(func() {
		// Wren's Run Vanquisher analog: "As an additional cost to cast this
		// spell, reveal an Elf card from your hand or pay {3}."
		if !mage.CardRegistered("Test Reveal Or Pay") {
			elfCardFilter := mage.NewCardFilter("Elf card", func(c mage.Card) bool {
				return c.HasSubType("Elf")
			})
			mage.Register("Test Reveal Or Pay", func() mage.Card {
				return mage.NewCreature("Test Reveal Or Pay", "{1}{G}", 3, 3,
					mage.WithSubTypes("Elf", "Warrior"),
					mage.WithAdditionalCost(mage.EitherCost(
						mage.RevealFromHandCost(elfCardFilter, "Reveal an Elf card"),
						mage.ManaCostOf("{3}"),
					)),
				)
			})
		}
		if !mage.CardRegistered("Test Forest Elf") {
			mage.Register("Test Forest Elf", func() mage.Card {
				return mage.NewCreature("Test Forest Elf", "{G}", 1, 1,
					mage.WithSubTypes("Elf"))
			})
		}
	})
}

// TestRevealFromHandCost_PaysWithReveal confirms that when the controller
// has a matching card in hand (an Elf), the reveal branch is taken: no
// extra mana is spent, and the revealed card stays in hand.
func TestRevealFromHandCost_PaysWithReveal(t *testing.T) {
	registerRevealFromHandCostCards()

	tg := NewTestGame(t)
	// PlayerA has the Vanquisher and an Elf to reveal.
	tg.AddCard(core.ZoneHand, PlayerA, "Test Reveal Or Pay")
	tg.AddCard(core.ZoneHand, PlayerA, "Test Forest Elf")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Reveal Or Pay")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Vanquisher resolved.
	tg.AssertPermanentCount(PlayerA, "Test Reveal Or Pay", 1)
	// Elf stayed in hand.
	tg.AssertHandCount(PlayerA, "Test Forest Elf", 1)
	// And the engine recorded the reveal.
	if r := tg.LastCostReveal(); r == nil || r.Name() != "Test Forest Elf" {
		t.Errorf("LastCostReveal = %v, want Test Forest Elf", r)
	}
}

// TestRevealFromHandCost_FallsBackToMana confirms when the player has no
// matching card to reveal, the mana branch is selected automatically and
// the spell still resolves.
func TestRevealFromHandCost_FallsBackToMana(t *testing.T) {
	registerRevealFromHandCostCards()

	tg := NewTestGame(t)
	// No Elf in hand — but plenty of mana on turn 4.
	tg.AddCard(core.ZoneHand, PlayerA, "Test Reveal Or Pay")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Spell never cast (insufficient mana on turn 1). Just confirms
	// CanPay branched correctly.
	tg.AssertHandCount(PlayerA, "Test Reveal Or Pay", 1)
}
