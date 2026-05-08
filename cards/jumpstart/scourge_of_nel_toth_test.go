package jumpstart

import (
	"testing"

	"github.com/google/uuid"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

func findCardInGraveyard(g *gametest.TestGame, ref gametest.PlayerRef, name string) uuid.UUID {
	pl := g.GetPlayer(ref)
	for _, c := range pl.Graveyard() {
		if c.Name() == name {
			return c.ID()
		}
	}
	return uuid.Nil
}

func TestScourgeOfNelToth_NormalCastFromHand(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Scourge of Nel Toth")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 7)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Scourge of Nel Toth")
	g.StopAt(1, core.EndStep)
	g.Execute()

	g.AssertPermanentCount(gametest.PlayerA, "Scourge of Nel Toth", 1)
	g.AssertPowerToughness(gametest.PlayerA, "Scourge of Nel Toth", 6, 6)
	g.AssertHasAbility(gametest.PlayerA, "Scourge of Nel Toth", core.Flying, true)
}

func TestScourgeOfNelToth_AlternateCostFromGraveyard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Scourge of Nel Toth")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 2)
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	pA := g.GetPlayer(gametest.PlayerA)
	pA.ManaPool().Clear()
	pA.ManaPool().Add(core.Black, 2)

	scourgeID := findCardInGraveyard(g, gametest.PlayerA, "Scourge of Nel Toth")
	if scourgeID == uuid.Nil {
		t.Fatalf("Scourge not in graveyard")
	}

	if err := g.CastCardWithAlternateCost(pA.PlayerID(), scourgeID, 0, nil, 0); err != nil {
		t.Fatalf("alt-cost cast: %v", err)
	}
	g.ResolveStack()

	g.AssertPermanentCount(gametest.PlayerA, "Scourge of Nel Toth", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 2)
	g.AssertGraveyardCount(gametest.PlayerA, "Scourge of Nel Toth", 0)
}

func TestScourgeOfNelToth_AlternateCostInsufficientMana(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Scourge of Nel Toth")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears", 2)
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	pA := g.GetPlayer(gametest.PlayerA)
	pA.ManaPool().Clear()
	pA.ManaPool().Add(core.Black, 1)

	scourgeID := findCardInGraveyard(g, gametest.PlayerA, "Scourge of Nel Toth")
	if scourgeID == uuid.Nil {
		t.Fatalf("Scourge not in graveyard")
	}

	if err := g.CastCardWithAlternateCost(pA.PlayerID(), scourgeID, 0, nil, 0); err == nil {
		t.Fatalf("expected alt-cost cast to fail with insufficient mana")
	}
	g.AssertPermanentCount(gametest.PlayerA, "Scourge of Nel Toth", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Scourge of Nel Toth", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 2)
}

func TestScourgeOfNelToth_AlternateCostInsufficientCreatures(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Scourge of Nel Toth")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears", 1)
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	pA := g.GetPlayer(gametest.PlayerA)
	pA.ManaPool().Clear()
	pA.ManaPool().Add(core.Black, 2)

	scourgeID := findCardInGraveyard(g, gametest.PlayerA, "Scourge of Nel Toth")
	if scourgeID == uuid.Nil {
		t.Fatalf("Scourge not in graveyard")
	}

	if err := g.CastCardWithAlternateCost(pA.PlayerID(), scourgeID, 0, nil, 0); err == nil {
		t.Fatalf("expected alt-cost cast to fail with only 1 creature available to sacrifice")
	}
	g.AssertPermanentCount(gametest.PlayerA, "Scourge of Nel Toth", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Scourge of Nel Toth", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestScourgeOfNelToth_RegistersAlternateCost(t *testing.T) {
	card, err := mage.CreateCard("Scourge of Nel Toth")
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	bc, ok := card.(*mage.BaseCard)
	if !ok {
		t.Fatalf("expected *BaseCard")
	}
	alts := bc.AlternateCosts()
	if len(alts) != 1 {
		t.Fatalf("expected 1 alternate cost, got %d", len(alts))
	}
	if alts[0].Zone != core.ZoneGraveyard {
		t.Errorf("expected ZoneGraveyard, got %v", alts[0].Zone)
	}
	if alts[0].Mana.Black != 2 {
		t.Errorf("expected {B}{B}, got %v", alts[0].Mana)
	}
	if len(alts[0].Additional) != 2 {
		t.Errorf("expected 2 additional costs (sacrifice two creatures), got %d", len(alts[0].Additional))
	}
}
