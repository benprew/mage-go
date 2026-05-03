package jumpstart

import (
	"testing"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// Mode 0: pick a creature in your graveyard and return it to your hand.
// Per CR 603.1f / 603.3d, the mode is chosen as the trigger goes on the
// stack, then the per-mode target is chosen at the same time.
func TestEntomberExarch_ReturnCreatureFromGraveyard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(ZoneHand, gametest.PlayerA, "Entomber Exarch")
	g.AddCard(ZoneGraveyard, gametest.PlayerA, "Hill Giant")
	g.ChooseMode(gametest.PlayerA, 0)
	g.ChooseTarget(gametest.PlayerA, "Hill Giant")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Entomber Exarch")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 0)
}

// Mode 0 with no creature card in graveyard: trigger still goes on the
// stack with the chosen mode but resolves to nothing (no legal target).
// CR 603.3c: a mode whose targets can't be legally chosen is still chosable;
// the ability resolves with no effect at the target step.
func TestEntomberExarch_Mode0NoCreaturesInGraveyard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(ZoneHand, gametest.PlayerA, "Entomber Exarch")
	g.ChooseMode(gametest.PlayerA, 0)
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Entomber Exarch")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Entomber Exarch", 1)
}

// Mode 1: target opponent reveals their hand; you pick a noncreature card;
// they discard it. Hill Giant (creature) must NOT be a legal pick.
func TestEntomberExarch_OpponentDiscardsNoncreature(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(ZoneHand, gametest.PlayerA, "Entomber Exarch")
	g.AddCard(ZoneHand, gametest.PlayerB, "Mountain")
	g.AddCard(ZoneHand, gametest.PlayerB, "Grizzly Bears")
	g.ChooseMode(gametest.PlayerA, 1)
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.ChooseFromLibrary(gametest.PlayerA, "Mountain")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Entomber Exarch")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Mountain", 1)
	g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 1)
}

// Mode 1 with only creatures in opponent's hand: no noncreature card to
// pick; the discard step is a no-op. Grizzly Bears must remain in hand.
func TestEntomberExarch_Mode1NoNoncreaturesInHand(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	g.AddCard(ZoneHand, gametest.PlayerA, "Entomber Exarch")
	g.AddCard(ZoneHand, gametest.PlayerB, "Grizzly Bears")
	g.ChooseMode(gametest.PlayerA, 1)
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.CastSpell(1, PrecombatMain, gametest.PlayerA, "Entomber Exarch")
	g.StopAt(1, PostcombatMain)
	g.Execute()
	g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 0)
}
