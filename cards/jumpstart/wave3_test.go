package jumpstart

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// TestThirstForKnowledge_DiscardArtifact verifies the controller defaults to
// "yes pay" and discards an artifact card to avoid the 2-card discard.
func TestThirstForKnowledge_DiscardArtifact(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Thirst for Knowledge")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains", 5)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sol Ring")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Thirst for Knowledge")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Sol Ring", 1)
}

// TestReadTheRunes_X1Discard verifies that for X=1 with the controller
// declining the sacrifice option, the discard branch fires once.
func TestReadTheRunes_X1Discard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Read the Runes")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
	tp := g.GetPlayer(gametest.PlayerA)
	tp.QueueMayAbilityChoices(false)
	g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Read the Runes", 1)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Read the Runes", 1)
}

// TestRhysticStudy_OpponentDeclinesPay verifies that when an opponent casts
// a spell and declines/cannot pay {1}, the controller draws a card.
func TestRhysticStudy_OpponentDeclinesPay(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rhystic Study")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.SetLife(gametest.PlayerA, 20)
	tpB := g.GetPlayer(gametest.PlayerB)
	tpB.QueueMayAbilityChoices(false)
	g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
	g.StopAt(2, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Plains", 1)
	g.AssertLife(gametest.PlayerA, 17)
}

// TestDraconicRoar_RevealedDragonDealsExtra verifies the optional reveal pays
// off as +3 damage to the targeted creature's controller.
func TestDraconicRoar_RevealedDragonDealsExtra(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerB, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Draconic Roar")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Shivan Dragon")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Draconic Roar", "Grizzly Bears")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertLife(gametest.PlayerB, 17)
	g.AssertHandCount(gametest.PlayerA, "Shivan Dragon", 1)
}

// TestDraconicRoar_NoDragonNoBonus verifies that without a Dragon to reveal
// or control, only the base 3 damage applies and the controller takes no hit.
func TestDraconicRoar_NoDragonNoBonus(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.SetLife(gametest.PlayerB, 20)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Draconic Roar")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Draconic Roar", "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
	g.AssertLife(gametest.PlayerB, 20)
}

// TestExplore_AllowsSecondLand verifies that after Explore resolves, the
// active player's land-play allowance is bumped to 2. The harness's auto-
// land-play loop then plays both Forests in the same main phase.
func TestExplore_AllowsSecondLand(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Explore")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Explore")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Forest", 4)
}
