package jumpstart

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

// Etali attacks; the top card of each player's library is exiled. Both end
// up in the exile zone before any cast decisions are made.
func TestEtaliPrimalStorm_ExilesTopOfEachLibrary(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Etali, Primal Storm")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Swamp")
	g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(false, false)
	g.Attack(1, gametest.PlayerA, "Etali, Primal Storm")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertExileCount("Plains", 1)
	g.AssertExileCount("Swamp", 1)
}

// Opponent-owned card on top of opponent's library can be cast by Etali's
// controller without paying mana. The Lightning Bolt resolves under
// PlayerA's control: PlayerA picks the target (PlayerB) and PlayerB takes
// 3 damage. Per CR 706.10, owner determines graveyard placement, so the
// resolved Lightning Bolt ends up in PlayerB's graveyard.
func TestEtaliPrimalStorm_CastsOpponentOwnedExiledCard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Etali, Primal Storm")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Lightning Bolt")
	g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(true)
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.Attack(1, gametest.PlayerA, "Etali, Primal Storm")
	g.StopAt(1, core.EndStep)
	g.Execute()

	// 6 combat damage from Etali + 3 from Lightning Bolt cast by PlayerA.
	g.AssertLife(gametest.PlayerB, 20-6-3)
	g.AssertGraveyardCount(gametest.PlayerB, "Lightning Bolt", 1)
	g.AssertExileCount("Lightning Bolt", 0)
}

// PlayerA may decline some exiled cards and cast others. Here PlayerA
// declines their own (a Plains, which is anyway uncastable as a land) and
// casts the opponent-owned Lightning Bolt.
func TestEtaliPrimalStorm_CastsOnlyOpponentCardDeclinesOwn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Etali, Primal Storm")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Shock")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Lightning Bolt")
	// First decision: decline PlayerA's own Shock. Second: accept Lightning Bolt.
	g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(false, true)
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.Attack(1, gametest.PlayerA, "Etali, Primal Storm")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertLife(gametest.PlayerB, 20-6-3)
	g.AssertExileCount("Shock", 1)
	g.AssertExileCount("Lightning Bolt", 0)
	g.AssertGraveyardCount(gametest.PlayerB, "Lightning Bolt", 1)
}

// Lands can't be cast as spells. An exiled land stays in exile even when
// PlayerA would otherwise want to play it.
func TestEtaliPrimalStorm_LandsStayInExile(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Etali, Primal Storm")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest")
	g.Attack(1, gametest.PlayerA, "Etali, Primal Storm")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertExileCount("Mountain", 1)
	g.AssertExileCount("Forest", 1)
}
