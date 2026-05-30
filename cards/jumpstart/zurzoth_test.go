package jumpstart

import (
	"math/rand"
	"testing"

	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

// Zurzoth, Chaos Rider {2}{R}
// Legendary Creature — Devil  2/3
// Whenever an opponent draws their first card each turn, if it's not their
// turn, you create a 1/1 red Devil creature token with "When this token dies,
// it deals 1 damage to any target."
// Whenever one or more Devils you control attack one or more players, you and
// those players each draw a card, then discard a card at random.

// Opponent's first card draw on your turn → token created.
func TestZurzoth_OpponentDrawsOnYourTurn_CreatesToken(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Zurzoth, Chaos Rider")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Ancestral Recall")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 1)
	for range 5 {
		g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest")
	}
	// PlayerA casts Ancestral Recall targeting PlayerB during PlayerA's turn.
	// PlayerB will draw 3 cards. The first triggers Zurzoth (it's not
	// PlayerB's turn, and it's their first card this turn). The 2nd and 3rd
	// must NOT trigger.
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ancestral Recall", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Devil", 1)
}

// Opponent's draw on their own turn → no token (it IS their turn).
func TestZurzoth_OpponentDrawsOnTheirOwnTurn_NoToken(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Zurzoth, Chaos Rider")
	for range 5 {
		g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest")
	}
	// Run through PlayerB's turn-2 draw step (their normal draw). Since
	// it IS PlayerB's turn, no Devil token should be created.
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Devil", 0)
}

// Controller's own draw → never triggers (drawing player is not an opponent).
func TestZurzoth_ControllerDraws_NoToken(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Zurzoth, Chaos Rider")
	for range 5 {
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")
	}
	// PlayerA's turn 3 draw step (turn 1 has no draw for first player; pick
	// turn 3 just to be safe). Drawer is the controller — not an opponent.
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Devil", 0)
}

// Multiple opponent draws on your turn → only first triggers (one token).
func TestZurzoth_MultipleOpponentDraws_OnlyFirstTriggers(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Zurzoth, Chaos Rider")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Ancestral Recall", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
	for range 8 {
		g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest")
	}
	// Two Recalls each draw 3 for PlayerB on PlayerA's turn = 6 draws total,
	// but only the very first card drawn this turn fires Zurzoth's trigger.
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ancestral Recall", "PlayerB")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Ancestral Recall", "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Devil", 1)
}

// Devils-attack trigger: one Devil attacker → both controller and defender
// draw 1 and discard 1 at random.
func TestZurzoth_DevilAttacksTriggers_BothPlayersDrawAndDiscard(t *testing.T) {
	rand.Seed(1) //nolint:staticcheck // SA1019: tests rely on global rand seeding for determinism
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Zurzoth, Chaos Rider")
	// Give each player an extra hand card so the random discard has
	// something to choose; library lets the draw succeed.
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mountain", 1)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Forest", 1)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 5)
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest", 5)
	g.Attack(3, gametest.PlayerA, "Zurzoth, Chaos Rider")
	g.StopAt(3, core.EndStep)
	g.Execute()
	// Each player drew one and discarded one at random, so each graveyard
	// holds exactly one card. Both hands are basics-only, so the discard is
	// always a Mountain (PlayerA) or Forest (PlayerB).
	g.AssertGraveyardCount(gametest.PlayerA, "Mountain", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Forest", 1)
}

// Non-Devil attacker → clause 2 does not trigger.
func TestZurzoth_NonDevilAttacker_NoDrawDiscard(t *testing.T) {
	rand.Seed(2) //nolint:staticcheck // SA1019: tests rely on global rand seeding for determinism
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Zurzoth, Chaos Rider")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mountain", 1)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Forest", 1)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain", 5)
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest", 5)
	// Attack only with the Bears (not a Devil). No draw/discard.
	g.Attack(3, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(3, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Mountain", 0)
	g.AssertGraveyardCount(gametest.PlayerB, "Forest", 0)
}
