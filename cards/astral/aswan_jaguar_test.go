package astral

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestAswanJaguarChoosesRandomTypeFromTargetOpponentsLibrary(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Prismatic Dragon")
	g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Rainbow Knights")
	g.ChooseTarget(gametest.PlayerA, "PlayerB")
	g.SetRandomResults([]int{1})
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Aswan Jaguar")
	g.ResolveStack()

	jaguar := g.FindPermanentByName("Aswan Jaguar", g.GetPlayer(gametest.PlayerA).PlayerID())
	if jaguar == nil {
		t.Fatal("Aswan Jaguar not found")
	}
	if jaguar.ChosenSubtype != "Knights" {
		t.Fatalf("chosen subtype = %q, want Knights", jaguar.ChosenSubtype)
	}
}

func TestAswanJaguarBuriesOnlyTheChosenType(t *testing.T) {
	g := gametest.NewTestGame(t)
	jaguarID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Aswan Jaguar")
	dragonID := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Prismatic Dragon")
	knightsID := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Rainbow Knights")
	jaguar := g.FindPermanent(jaguarID)
	jaguar.ChosenSubtype = "Dragon"
	player := g.GetPlayer(gametest.PlayerA)
	player.ManaPool().Add(core.Green, 2)

	if err := g.ActivateAbilityByIndex(player.PlayerID(), jaguarID, 1, []uuid.UUID{knightsID}); err == nil {
		t.Fatal("activation accepted a creature without the chosen subtype")
	}
	if jaguar.Tapped {
		t.Fatal("failed activation tapped Aswan Jaguar")
	}
	if err := g.ActivateAbilityByIndex(player.PlayerID(), jaguarID, 1, []uuid.UUID{dragonID}); err != nil {
		t.Fatalf("activate Aswan Jaguar: %v", err)
	}
	g.ResolveStack()

	g.AssertPermanentCount(gametest.PlayerB, "Prismatic Dragon", 0)
	g.AssertGraveyardCount(gametest.PlayerB, "Prismatic Dragon", 1)
}
