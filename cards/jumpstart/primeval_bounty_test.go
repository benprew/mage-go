package jumpstart

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

// Casting a creature spell triggers the first ability and creates a 3/3
// green Beast token in addition to the cast creature resolving.
func TestPrimevalBounty_CastCreatureMakesBeastToken(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Primeval Bounty")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Beast", 1)

	pid := g.GetPlayer(gametest.PlayerA).PlayerID()
	beast := g.FindPermanentByName("Beast", pid)
	if beast == nil {
		t.Fatalf("expected a Beast token on the battlefield")
	}
	if beast.Card.Power() != 3 || beast.Card.Toughness() != 3 {
		t.Errorf("Beast token P/T: got %d/%d want 3/3",
			beast.Card.Power(), beast.Card.Toughness())
	}
	hasGreen := false
	for _, c := range beast.Colors() {
		if c == core.Green {
			hasGreen = true
		}
	}
	if !hasGreen {
		t.Errorf("Beast token should be green; got colors %v", beast.Colors())
	}
}

// Casting a noncreature spell triggers the second ability; the controller
// chooses a creature they control as the target and it gets three +1/+1
// counters. Casting a creature spell does NOT trigger this ability.
func TestPrimevalBounty_CastNoncreatureAddsCountersToTarget(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Primeval Bounty")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
	g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 17)
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 3)
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 5, 5)
}

// Casting a creature spell does NOT trigger the noncreature clause. Only
// a Beast token appears; no counters are applied.
func TestPrimevalBounty_CastCreatureDoesNotAddCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Primeval Bounty")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Beast", 1)
	g.AssertCounterCount(gametest.PlayerA, "Grizzly Bears", core.P1P1, 0)
}

// Landfall: when a land you control enters, you gain 3 life.
func TestPrimevalBounty_LandfallGainsThreeLife(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Primeval Bounty")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")

	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	pa := g.GetPlayer(gametest.PlayerA)
	pid := pa.PlayerID()
	var forest mage.Card
	for _, c := range pa.Hand() {
		if c.Name() == "Forest" {
			forest = c
			break
		}
	}
	if forest == nil {
		t.Fatalf("expected Forest in hand")
	}
	if err := g.Game.PlayLand(pid, forest.ID()); err != nil {
		t.Fatalf("PlayLand: %v", err)
	}
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 23)
}

// Landfall does NOT trigger when an opponent's land enters.
func TestPrimevalBounty_OpponentLandDoesNotTrigger(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Primeval Bounty")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Forest")

	g.StopAt(2, core.PrecombatMain)
	g.Execute()

	pb := g.GetPlayer(gametest.PlayerB)
	pid := pb.PlayerID()
	var forest mage.Card
	for _, c := range pb.Hand() {
		if c.Name() == "Forest" {
			forest = c
			break
		}
	}
	if forest == nil {
		t.Fatalf("expected Forest in PlayerB's hand")
	}
	if err := g.Game.PlayLand(pid, forest.ID()); err != nil {
		t.Fatalf("PlayLand: %v", err)
	}
	g.StopAt(2, core.PostcombatMain)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 20)
}
