package jumpstart

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

// Oracle of Mul Daya {3}{G}
// Creature — Elf Shaman
// 2/2
// You may play an additional land on each of your turns.
// Play with the top card of your library revealed.
// You may play lands from the top of your library.

// TestOracleOfMulDaya_AdditionalLandPlay verifies that with Oracle of Mul
// Daya on the battlefield, the controller's MaxLandPlays is 2 and they can
// play two lands from hand on a single turn.
func TestOracleOfMulDaya_AdditionalLandPlay(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Oracle of Mul Daya")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest", 2)

	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	if got := g.MaxLandPlays(); got != 2 {
		t.Errorf("MaxLandPlays with Oracle: got %d want 2", got)
	}

	pid := g.GetPlayer(gametest.PlayerA).PlayerID()
	hand := g.GetPlayer(gametest.PlayerA).Hand()
	if len(hand) < 2 {
		t.Fatalf("expected 2 forests in hand, got %d", len(hand))
	}
	if err := g.Game.PlayLand(pid, hand[0].ID()); err != nil {
		t.Fatalf("first land play: %v", err)
	}
	if err := g.Game.PlayLand(pid, hand[1].ID()); err != nil {
		t.Fatalf("second land play: %v", err)
	}
	g.AssertPermanentCount(gametest.PlayerA, "Forest", 2)
}

// TestOracleOfMulDaya_TopCardRevealed verifies the controller's library top
// is marked revealed while Oracle is on the battlefield, and the flag
// clears when Oracle leaves play.
func TestOracleOfMulDaya_TopCardRevealed(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Oracle of Mul Daya")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")

	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	pid := g.GetPlayer(gametest.PlayerA).PlayerID()
	if !g.IsTopCardRevealed(pid) {
		t.Errorf("expected top card revealed for PlayerA while Oracle is on battlefield")
	}

	oracle := g.FindPermanentByName("Oracle of Mul Daya", pid)
	if oracle == nil {
		t.Fatal("Oracle of Mul Daya not found on battlefield")
	}
	g.DestroyPermanent(oracle)
	g.ApplyEffects()

	if g.IsTopCardRevealed(pid) {
		t.Errorf("expected top card no longer revealed after Oracle leaves play")
	}
}

// TestOracleOfMulDaya_PlayLandFromTop verifies the controller can play a
// land from the top of their library while Oracle is on the battlefield.
func TestOracleOfMulDaya_PlayLandFromTop(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Oracle of Mul Daya")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")

	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	pid := g.GetPlayer(gametest.PlayerA).PlayerID()
	if !g.CanPlayLandsFromZone(pid, core.ZoneLibrary) {
		t.Fatal("expected play-lands-from-library permission active")
	}

	pl := g.GetPlayer(gametest.PlayerA)
	libBefore := len(pl.Library())
	top := pl.Library()[0]
	if top.Name() != "Mountain" {
		t.Fatalf("expected top of library Mountain, got %s", top.Name())
	}

	if err := g.Game.PlayLand(pid, top.ID()); err != nil {
		t.Fatalf("PlayLand from library top: %v", err)
	}

	if got := len(pl.Library()); got != libBefore-1 {
		t.Errorf("library size: got %d want %d", got, libBefore-1)
	}
	g.AssertPermanentCount(gametest.PlayerA, "Mountain", 1)
}

// TestOracleOfMulDaya_CantPlayLandWhenAtMax verifies that once the
// controller has played their full land allowance (1 from hand + 1 from
// top under Oracle = 2), a third attempt is rejected.
func TestOracleOfMulDaya_CantPlayLandWhenAtMax(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Oracle of Mul Daya")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Plains")

	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	pid := g.GetPlayer(gametest.PlayerA).PlayerID()
	pl := g.GetPlayer(gametest.PlayerA)

	forest := pl.Hand()[0]
	if err := g.Game.PlayLand(pid, forest.ID()); err != nil {
		t.Fatalf("first land (Forest from hand): %v", err)
	}

	top := pl.Library()[0]
	if err := g.Game.PlayLand(pid, top.ID()); err != nil {
		t.Fatalf("second land (top of library): %v", err)
	}

	nextTop := pl.Library()[0]
	if err := g.Game.PlayLand(pid, nextTop.ID()); err == nil {
		t.Errorf("expected error playing third land in one turn, got nil")
	}
}
