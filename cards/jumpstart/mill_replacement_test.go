package jumpstart

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// TestBruvac_DoublesOpponentMill verifies that while Bruvac is on the
// battlefield under your control, opponents who are milled mill twice as
// many cards.
func TestBruvac_DoublesOpponentMill(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Island", 5)
	g.AddCard(ZoneHand, gametest.PlayerA, "Bruvac the Grandiloquent")
	g.AddCard(ZoneHand, gametest.PlayerA, "Thought Scour")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Bruvac the Grandiloquent")
	g.CastSpell(1, PostcombatMain, gametest.PlayerA, "Thought Scour", "PlayerB")
	g.StopAt(1, EndStep)
	g.Execute()
	// Thought Scour mills 2 → doubled to 4 of PlayerB's padded library cards.
	g.AssertGraveyardCount(gametest.PlayerB, "Filler", 4)
}

// TestBruvac_DoesNotDoubleSelfMill verifies the modifier does not apply
// when Bruvac's controller is the milled player.
func TestBruvac_DoesNotDoubleSelfMill(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Island", 5)
	g.AddCard(ZoneHand, gametest.PlayerA, "Bruvac the Grandiloquent")
	g.AddCard(ZoneHand, gametest.PlayerA, "Thought Scour")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Bruvac the Grandiloquent")
	g.CastSpell(1, PostcombatMain, gametest.PlayerA, "Thought Scour", "PlayerA")
	g.StopAt(1, EndStep)
	g.Execute()
	// Self-mill is not doubled: 2 cards from PlayerA's library.
	g.AssertGraveyardCount(gametest.PlayerA, "Filler", 2)
}
