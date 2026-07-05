package legends

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

// TestHammerheimRemovesLandwalkFromMultipleLords verifies that Hammerheim's
// "loses all landwalk" removes mountainwalk from a Goblin even when two Goblin
// Kings are granting it. Landwalk instances are redundant (CR 702.14e), and the
// later-timestamped removal wins (CR 613.9), so the creature becomes blockable.
func TestHammerheimRemovesLandwalkFromMultipleLords(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hammerheim")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Goblin King")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Goblin King")
	raidersID := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mons's Goblin Raiders")

	g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerA, "Hammerheim", "Mons's Goblin Raiders")
	g.StopAt(2, core.EndStep)
	g.Execute()

	perm := g.FindPermanent(raidersID)
	if perm == nil {
		t.Fatal("raiders not found")
	}
	if perm.HasKeyword(core.Mountainwalk) {
		t.Error("expected Mons's Goblin Raiders to lose mountainwalk after Hammerheim removed all landwalk (two Goblin Kings granting)")
	}
}
