package jumpstart

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

// TestSanitariumSkeleton_ReturnFromGraveyard verifies the
// graveyard-activated ability returns the Skeleton to its owner's hand.
func TestSanitariumSkeleton_ReturnFromGraveyard(t *testing.T) {
	tg := gametest.NewTestGame(t)
	tg.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Sanitarium Skeleton")
	tg.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	pid := tg.GetPlayer(gametest.PlayerA).PlayerID()

	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	skel, idx, ok := tg.FindGraveyardActivatableCard(pid, "Sanitarium Skeleton")
	if !ok {
		t.Fatal("graveyard activatable Sanitarium Skeleton not found")
	}
	if err := tg.ActivateGraveyardAbility(pid, skel, idx, nil); err != nil {
		t.Fatalf("ActivateGraveyardAbility: %v", err)
	}
	tg.ResolveStack()

	tg.AssertHandCount(gametest.PlayerA, "Sanitarium Skeleton", 1)
	tg.AssertGraveyardCount(gametest.PlayerA, "Sanitarium Skeleton", 0)
}

// TestGhoulcallersAccomplice_ExileToCreateZombie verifies the
// graveyard-activated ability exiles the source and creates a 2/2
// black Zombie token.
func TestGhoulcallersAccomplice_ExileToCreateZombie(t *testing.T) {
	tg := gametest.NewTestGame(t)
	tg.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Ghoulcaller's Accomplice")
	tg.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 4)
	pid := tg.GetPlayer(gametest.PlayerA).PlayerID()

	tg.StopAt(1, core.PrecombatMain)
	tg.Execute()

	acc, idx, ok := tg.FindGraveyardActivatableCard(pid, "Ghoulcaller's Accomplice")
	if !ok {
		t.Fatal("graveyard activatable Ghoulcaller's Accomplice not found")
	}
	if err := tg.ActivateGraveyardAbility(pid, acc, idx, nil); err != nil {
		t.Fatalf("ActivateGraveyardAbility: %v", err)
	}
	tg.ResolveStack()

	tg.AssertExileCount("Ghoulcaller's Accomplice", 1)
	tg.AssertGraveyardCount(gametest.PlayerA, "Ghoulcaller's Accomplice", 0)
	tg.AssertPermanentCount(gametest.PlayerA, "Zombie", 1)
}

// TestCauldronFamiliar_SacFoodReturnsFromGraveyard verifies the Familiar's
// graveyard ability: Sacrifice a Food returns it to the battlefield.
func TestCauldronFamiliar_SacFoodReturnsFromGraveyard(t *testing.T) {
	tg := gametest.NewTestGame(t)
	tg.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Cauldron Familiar")
	// Bake into a Pie creates a Food on resolution.
	tg.AddCard(core.ZoneHand, gametest.PlayerA, "Bake into a Pie")
	tg.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
	tg.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	tg.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Bake into a Pie", "Grizzly Bears")
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()

	tg.AssertPermanentCount(gametest.PlayerA, "Food", 1)

	pid := tg.GetPlayer(gametest.PlayerA).PlayerID()
	cat, idx, ok := tg.FindGraveyardActivatableCard(pid, "Cauldron Familiar")
	if !ok {
		t.Fatal("graveyard activatable Cauldron Familiar not found")
	}
	if err := tg.ActivateGraveyardAbility(pid, cat, idx, nil); err != nil {
		t.Fatalf("ActivateGraveyardAbility: %v", err)
	}
	tg.ResolveStack()

	tg.AssertGraveyardCount(gametest.PlayerA, "Cauldron Familiar", 0)
	tg.AssertPermanentCount(gametest.PlayerA, "Cauldron Familiar", 1)
	tg.AssertPermanentCount(gametest.PlayerA, "Food", 0)
}
