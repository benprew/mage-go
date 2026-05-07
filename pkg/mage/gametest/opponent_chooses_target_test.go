package gametest

import (
	"sync"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

var opponentChoosesOnce sync.Once

func registerOpponentChoosesTargetCards() {
	opponentChoosesOnce.Do(func() {
		// Mausoleum Turnkey analog: ETB returns target creature card of
		// an opponent's choice from your graveyard to your hand.
		if !mage.CardRegistered("Test Opponent Picks Bear") {
			mage.Register("Test Opponent Picks Bear", func() mage.Card {
				return mage.NewCreature("Test Opponent Picks Bear", "{3}{B}", 3, 2,
					mage.WithSubTypes("Ogre"),
					mage.WithAbility(mage.EntersBattlefieldTrigger(
						mage.ReturnFromGraveyardToHandTarget(),
						false,
					).AddTarget(mage.TargetOpponentChoice(
						mage.TargetCardInYourGraveyard(mage.IsCreatureCard),
					))),
				)
			})
		}
		if !mage.CardRegistered("Test Filler Bear") {
			mage.Register("Test Filler Bear", func() mage.Card {
				return mage.NewCreature("Test Filler Bear", "{1}{G}", 2, 2,
					mage.WithSubTypes("Bear"))
			})
		}
	})
}

// TestOpponentChoosesTarget_PromptsOpponent verifies that wrapping a
// target in TargetOpponentChoice routes the ChooseTargets prompt to the
// opposing player rather than the trigger's controller. We confirm by
// scripting the *opponent's* (PlayerB's) target choice — if the prompt
// went to PlayerA the script would not be consumed and PlayerA's
// fallback behavior would pick the first match.
func TestOpponentChoosesTarget_PromptsOpponent(t *testing.T) {
	registerOpponentChoosesTargetCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Swamp", 4)
	tg.AddCard(core.ZoneHand, PlayerA, "Test Opponent Picks Bear")
	// Two creature cards in A's graveyard. We script B to pick "Test Filler Bear".
	tg.AddCard(core.ZoneGraveyard, PlayerA, "Test Filler Bear")
	tg.AddCard(core.ZoneGraveyard, PlayerA, "Grizzly Bears")
	// Script B's choice. Also script A — if A is wrongly prompted, A would
	// pick "Grizzly Bears", which makes the assertion fail.
	tg.ChooseTarget(PlayerB, "Test Filler Bear")
	tg.ChooseTarget(PlayerA, "Grizzly Bears")

	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Opponent Picks Bear")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// PlayerB's scripted "Test Filler Bear" must have been the chosen
	// target — proving the prompt was routed to the opponent.
	tg.AssertHandCount(PlayerA, "Test Filler Bear", 1)
	tg.AssertGraveyardCount(PlayerA, "Test Filler Bear", 0)
	// And "Grizzly Bears" stayed in the graveyard.
	tg.AssertGraveyardCount(PlayerA, "Grizzly Bears", 1)
}
