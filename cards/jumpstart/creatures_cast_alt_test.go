package jumpstart

import (
	"testing"

	"github.com/google/uuid"

	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestScholarOfTheLostTrove_CastsInstantFromGraveyardForFree(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scholar of the Lost Trove")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertLife(gametest.PlayerB, 17)
	g.AssertExileCount("Lightning Bolt", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Lightning Bolt", 0)
}

func TestEtaliPrimalStorm_ExilesAndOptionallyCastsControllerLibraryCard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Etali, Primal Storm")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Plains")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.Attack(1, gametest.PlayerA, "Etali, Primal Storm")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertLife(gametest.PlayerB, 20-6-3)
}

func TestGontiLordOfLuxury_ExilesOpponentTopCardAndAllowsCast(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 5)
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gonti, Lord of Luxury")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.ChooseFromLibrary(gametest.PlayerA, "Lightning Bolt")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertExileCount("Lightning Bolt", 1)

	pA := g.GetPlayer(gametest.PlayerA)
	var foundID uuid.UUID
	for _, ec := range g.GetExile() {
		if ec.Card.Name() == "Lightning Bolt" {
			foundID = ec.Card.ID()
		}
	}
	if foundID == uuid.Nil {
		t.Fatalf("no exiled Lightning Bolt found")
	}
	if g.CastFromExilePermissionFor(pA.PlayerID(), foundID) == nil {
		t.Fatalf("expected cast-from-exile permission for Gonti's exiled card")
	}
}

func TestMaelstromArchangel_CastsFromHandOnCombatDamage(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Maelstrom Archangel")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.Attack(1, gametest.PlayerA, "Maelstrom Archangel")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertLife(gametest.PlayerB, 20-5-3)
	g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Lightning Bolt", 1)
}
