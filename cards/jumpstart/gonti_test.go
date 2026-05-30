package jumpstart

import (
	"testing"

	"github.com/google/uuid"

	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

// findGontiExiledCard returns the (exile-entry pointer, cardID) of the first
// exiled card carrying Gonti's cast-from-exile permission for playerID.
func findGontiExiledCard(t *testing.T, g *gametest.TestGame, playerID uuid.UUID) (*mage.ExiledCard, uuid.UUID) {
	t.Helper()
	exile := g.GetExile()
	for i := range exile {
		ec := &exile[i]
		if g.CastFromExilePermissionFor(playerID, ec.Card.ID()) != nil {
			return ec, ec.Card.ID()
		}
	}
	t.Fatalf("no exiled card with Gonti permission for player %s", playerID)
	return nil, uuid.Nil
}

func TestGontiLordOfLuxury_LooksAtTopFourAndExilesOneFaceDown(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Mountain")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gonti, Lord of Luxury")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.ChooseFromLibrary(gametest.PlayerA, "Lightning Bolt")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertExileCount("Lightning Bolt", 1)

	pA := g.GetPlayer(gametest.PlayerA)
	pB := g.GetPlayer(gametest.PlayerB)

	ec, _ := findGontiExiledCard(t, g, pA.PlayerID())
	if !ec.FaceDown {
		t.Errorf("expected exiled Lightning Bolt to be face down")
	}
	if !ec.VisibleTo(pA.PlayerID()) {
		t.Errorf("expected Gonti's controller to see the face-down card")
	}
	if ec.VisibleTo(pB.PlayerID()) {
		t.Errorf("expected opponent NOT to see the face-down card")
	}

	// Three remaining of the original four should be at the bottom of B's
	// library. The library starts with PlayerB's normal deck plus our four
	// stacked on top; after exiling one and sending the other three to the
	// bottom, PlayerB's library size is original - 1.
	startSize := pB.Library() // sanity: at least 3 left
	if len(startSize) < 3 {
		t.Fatalf("expected at least 3 cards remaining in PlayerB library, got %d", len(startSize))
	}
}

func TestGontiLordOfLuxury_ControllerCastsExiledCardWithMatchingMana(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gonti, Lord of Luxury")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.ChooseFromLibrary(gametest.PlayerA, "Lightning Bolt")
	g.StopAt(1, core.EndStep)
	g.Execute()

	pA := g.GetPlayer(gametest.PlayerA)
	pB := g.GetPlayer(gametest.PlayerB)

	_, cardID := findGontiExiledCard(t, g, pA.PlayerID())

	pA.ManaPool().Clear()
	pA.ManaPool().Add(core.Red, 1)

	if err := g.CastExiledCardWithPermission(pA.PlayerID(), cardID, []uuid.UUID{pB.PlayerID()}, 0); err != nil {
		t.Fatalf("CastExiledCardWithPermission: %v", err)
	}
	g.ResolveStack()

	if pB.Life() != 17 {
		t.Errorf("expected PlayerB life 17 after Bolt, got %d", pB.Life())
	}
	// Card no longer in exile, and permission is cleared.
	if g.FindExiledCard(cardID) != nil {
		t.Errorf("expected card removed from exile after cast")
	}
	if g.CastFromExilePermissionFor(pA.PlayerID(), cardID) != nil {
		t.Errorf("expected permission cleared after cast")
	}
}

func TestGontiLordOfLuxury_ControllerCastsWithAnyColorMana(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gonti, Lord of Luxury")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.ChooseFromLibrary(gametest.PlayerA, "Lightning Bolt")
	g.StopAt(1, core.EndStep)
	g.Execute()

	pA := g.GetPlayer(gametest.PlayerA)
	pB := g.GetPlayer(gametest.PlayerB)

	_, cardID := findGontiExiledCard(t, g, pA.PlayerID())

	// Lightning Bolt costs {R}; pay it with white mana via the any-type clause.
	pA.ManaPool().Clear()
	pA.ManaPool().Add(core.White, 1)

	if err := g.CastExiledCardWithPermission(pA.PlayerID(), cardID, []uuid.UUID{pB.PlayerID()}, 0); err != nil {
		t.Fatalf("CastExiledCardWithPermission with any-color mana: %v", err)
	}
	g.ResolveStack()

	if pB.Life() != 17 {
		t.Errorf("expected PlayerB life 17 after Bolt, got %d", pB.Life())
	}
}

func TestGontiLordOfLuxury_OpponentCannotCastExiledCard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Lightning Bolt")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gonti, Lord of Luxury")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.ChooseFromLibrary(gametest.PlayerA, "Lightning Bolt")
	g.StopAt(1, core.EndStep)
	g.Execute()

	pA := g.GetPlayer(gametest.PlayerA)
	pB := g.GetPlayer(gametest.PlayerB)

	_, cardID := findGontiExiledCard(t, g, pA.PlayerID())

	// PlayerB has plenty of red, but no permission was granted to them.
	pB.ManaPool().Clear()
	pB.ManaPool().Add(core.Red, 5)

	if err := g.CastExiledCardWithPermission(pB.PlayerID(), cardID, []uuid.UUID{pA.PlayerID()}, 0); err == nil {
		t.Fatalf("expected error: opponent has no permission to cast the exiled card")
	}
	if g.FindExiledCard(cardID) == nil {
		t.Errorf("expected card to remain in exile after rejected cast")
	}
}
