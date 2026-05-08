package gametest

import (
	"sync"
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

var millReplOnce sync.Once

func registerMillReplacementCards() {
	millReplOnce.Do(func() {
		// "If an opponent would mill one or more cards, they mill twice that
		// many cards instead." Bruvac-style.
		if !mage.CardRegistered("Test Mill Doubler") {
			mage.Register("Test Mill Doubler", func() mage.Card {
				return mage.NewCreature("Test Mill Doubler", "{2}{U}", 1, 4,
					mage.WithSubTypes("Human", "Advisor"),
					mage.WithETBEffect(mage.FuncEffect(
						"register mill doubler",
						mage.EffectProperties{},
						func(g *mage.Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
							g.AddMillReplacement(sourceID, func(g *mage.Game, milledID uuid.UUID, amt int) int {
								if milledID == controller {
									return amt
								}
								return amt * 2
							})
							return nil
						})),
				)
			})
		}
		// Test mill spell that mills 3.
		if !mage.CardRegistered("Test Mill 3") {
			mage.Register("Test Mill 3", func() mage.Card {
				return mage.NewSorcery("Test Mill 3", "{1}{U}",
					mage.NewTargetedSpell(mage.TargetPlayer(),
						mage.MillTargetPlayer(mage.Fixed(3))),
				)
			})
		}
	})
}

// TestMillReplacement_DoublesOpponent verifies the mill modifier doubles
// mill amounts when the milled player is an opponent of the source's
// controller.
func TestMillReplacement_DoublesOpponent(t *testing.T) {
	registerMillReplacementCards()

	tg := NewTestGame(t)
	doubler := tg.AddCard(core.ZoneBattlefield, PlayerA, "Test Mill Doubler")
	playerA := tg.getPlayerID(PlayerA)
	tg.AddMillReplacement(doubler, func(g *mage.Game, milledID uuid.UUID, amt int) int {
		if milledID == playerA {
			return amt
		}
		return amt * 2
	})
	tg.AddCard(core.ZoneHand, PlayerA, "Test Mill 3")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Mill 3", "PlayerB")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertGraveyardCount(PlayerB, "Filler", 6)
}

// TestMillReplacement_SkipsController verifies the doubling does not apply
// when the controller of the source is the one milling.
func TestMillReplacement_SkipsController(t *testing.T) {
	registerMillReplacementCards()

	tg := NewTestGame(t)
	doubler := tg.AddCard(core.ZoneBattlefield, PlayerA, "Test Mill Doubler")
	playerA := tg.getPlayerID(PlayerA)
	tg.AddMillReplacement(doubler, func(g *mage.Game, milledID uuid.UUID, amt int) int {
		if milledID == playerA {
			return amt
		}
		return amt * 2
	})
	tg.AddCard(core.ZoneHand, PlayerA, "Test Mill 3")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Mill 3", "PlayerA")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertGraveyardCount(PlayerA, "Filler", 3)
}
