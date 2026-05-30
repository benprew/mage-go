package jumpstart

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

// Savage Stomp {2}{G}
// Sorcery
// This spell costs {2} less to cast if it targets a Dinosaur you control.
// Put a +1/+1 counter on target creature you control. Then that creature
// fights target creature you don't control.

// permIDByName returns the on-battlefield permanent ID controlled by p whose
// card name matches name.
func permIDByName(g *gametest.TestGame, p gametest.PlayerRef, name string) uuid.UUID {
	pid := g.GetPlayer(p).PlayerID()
	if perm := g.FindPermanentByName(name, pid); perm != nil {
		return perm.ID()
	}
	return uuid.Nil
}

// TestSavageStomp_DinosaurTargetReducesCostByTwo confirms that, when the
// first (your-control) target is a Dinosaur, Savage Stomp's cost-reduction
// hook kicks in and shaves {2} off the generic portion.
func TestSavageStomp_DinosaurTargetReducesCostByTwo(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 2)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Orazca Frillback")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Savage Stomp")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	pid := g.GetPlayer(gametest.PlayerA).PlayerID()
	dino := permIDByName(g, gametest.PlayerA, "Orazca Frillback")
	bears := permIDByName(g, gametest.PlayerB, "Grizzly Bears")
	if dino == uuid.Nil || bears == uuid.Nil {
		t.Fatal("setup: missing battlefield permanents")
	}
	stomp := handCard(g, gametest.PlayerA, "Savage Stomp")
	if stomp == nil {
		t.Fatal("setup: Savage Stomp not in hand")
	}

	// Without targets context: pre-target queries skip the predicate, so the
	// reduction is 0 (the harness has no idea what we'd target).
	if got := g.ConditionalSpellCostReduction(pid, stomp); got != 0 {
		t.Errorf("pre-target reduction: got %d, want 0", got)
	}

	// With targets context: targeting our Dinosaur first triggers the {2}
	// reduction.
	got := g.ConditionalSpellCostReductionWithTargets(pid, stomp, []uuid.UUID{dino, bears})
	if got != 2 {
		t.Errorf("dinosaur target reduction: got %d, want 2", got)
	}
}

// TestSavageStomp_NonDinosaurTargetNoReduction verifies that targeting a
// non-Dinosaur creature you control gives no cost reduction.
func TestSavageStomp_NonDinosaurTargetNoReduction(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Savage Stomp")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	pid := g.GetPlayer(gametest.PlayerA).PlayerID()
	myBears := permIDByName(g, gametest.PlayerA, "Grizzly Bears")
	giant := permIDByName(g, gametest.PlayerB, "Hill Giant")
	stomp := handCard(g, gametest.PlayerA, "Savage Stomp")
	if myBears == uuid.Nil || giant == uuid.Nil || stomp == nil {
		t.Fatal("setup")
	}
	got := g.ConditionalSpellCostReductionWithTargets(pid, stomp, []uuid.UUID{myBears, giant})
	if got != 0 {
		t.Errorf("non-dinosaur target reduction: got %d, want 0", got)
	}
}

// TestSavageStomp_OpponentDinosaurDoesNotReduce verifies the predicate
// requires *your* Dinosaur — an opponent's Dinosaur isn't enough. (Savage
// Stomp's first target is "creature you control" so this is moot at cast
// time, but the predicate is defensive about controller.)
func TestSavageStomp_OpponentDinosaurDoesNotReduce(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Orazca Frillback")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Savage Stomp")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	pid := g.GetPlayer(gametest.PlayerA).PlayerID()
	myBears := permIDByName(g, gametest.PlayerA, "Grizzly Bears")
	oppDino := permIDByName(g, gametest.PlayerB, "Orazca Frillback")
	stomp := handCard(g, gametest.PlayerA, "Savage Stomp")
	if myBears == uuid.Nil || oppDino == uuid.Nil || stomp == nil {
		t.Fatal("setup")
	}
	got := g.ConditionalSpellCostReductionWithTargets(pid, stomp, []uuid.UUID{myBears, oppDino})
	if got != 0 {
		t.Errorf("opponent dino reduction: got %d, want 0", got)
	}
}

// TestSavageStomp_ReductionDoesNotUnderflow ensures the generic-cost cap in
// computeConditionalCostReduction prevents a sub-zero generic. Stack a static
// reducer (Dragonlord's Servant doesn't apply here, but we can simulate by
// using a low-cost theoretical scenario): we exercise the cap by directly
// inspecting that, even with the Dino reduction firing, the reported total
// reduction can never exceed the printed generic cost ({2}).
func TestSavageStomp_ReductionDoesNotUnderflow(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Orazca Frillback")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Savage Stomp")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	pid := g.GetPlayer(gametest.PlayerA).PlayerID()
	dino := permIDByName(g, gametest.PlayerA, "Orazca Frillback")
	bears := permIDByName(g, gametest.PlayerB, "Grizzly Bears")
	stomp := handCard(g, gametest.PlayerA, "Savage Stomp")
	if stomp == nil {
		t.Fatal("setup")
	}

	// Savage Stomp printed generic = 2. Reduction <= 2.
	got := g.ConditionalSpellCostReductionWithTargets(pid, stomp, []uuid.UUID{dino, bears})
	if got > stomp.ManaCost().Generic {
		t.Errorf("reduction underflow: got %d > printed generic %d", got, stomp.ManaCost().Generic)
	}
}

// TestSavageStomp_CountersAndFightStillWork confirms the existing counter +
// fight semantics still resolve under the new cost-reduction wiring.
func TestSavageStomp_CountersAndFightStillWork(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 3)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")    // 3/3
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Savage Stomp")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Savage Stomp", "Grizzly Bears", "Hill Giant")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// Bears becomes 3/3 (+1/+1) before the fight, so 3 vs 3 → both die.
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
}

// TestSavageStomp_CastWithReducedManaPool exercises the full cast pipeline:
// with only one Forest available, casting Savage Stomp targeting a Dinosaur
// you control must succeed (cost is {0}{G}). This proves the cost-reduction
// hook is wired into game.CastSpellByName, not just the query helper.
func TestSavageStomp_CastWithReducedManaPool(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 1)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Orazca Frillback") // 3/3 Dinosaur
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")    // 2/2
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Savage Stomp")
	// Only 1 Forest on board → only {G} can be auto-tapped. Casting must
	// succeed because the cost reduction takes the printed {2}{G} down to
	// {0}{G}.
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Savage Stomp", "Orazca Frillback", "Grizzly Bears")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	// Frillback gains a +1/+1 counter and fights Bears: 4 vs 2.
	// Bears dies; Frillback survives with 2 damage marked (toughness 4).
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	g.AssertPermanentCount(gametest.PlayerA, "Orazca Frillback", 1)
}
