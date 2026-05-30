package search

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestBlockerSubsets_RepairsSingleMenaceBlock(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Menace Bear", "{1}{R}", 2, 2, pa.PlayerID(), mage.WithKeyword(core.Menace))
	blk := makePerm("Grizzly Bears", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(atk, blk)
	g.GetCombat().AddAttacker(atk.ID(), pb.PlayerID())

	plans := blockerSubsets(g, pb.PlayerID())
	if len(plans) != 1 {
		t.Fatalf("blockerSubsets returned %d plans, want only no-block after menace repair", len(plans))
	}
	if len(plans[0]) != 0 {
		t.Fatalf("single menace block should be repaired away, got %v", plans[0])
	}
}

func TestBlockerSubsets_IncludesCanBlockAdditionalMultiBlock(t *testing.T) {
	g, pa, pb := makeGame()
	atk1 := makePerm("Attacker One", "{1}{R}", 2, 2, pa.PlayerID())
	atk2 := makePerm("Attacker Two", "{1}{R}", 2, 2, pa.PlayerID())
	blk := makePerm("Palace Guard", "{2}{W}", 1, 4, pb.PlayerID(), mage.WithKeyword(core.CanBlockAdditional))
	g.AddToBattlefield(atk1, atk2, blk)
	g.GetCombat().AddAttacker(atk1.ID(), pb.PlayerID())
	g.GetCombat().AddAttacker(atk2.ID(), pb.PlayerID())

	plans := blockerSubsets(g, pb.PlayerID())
	for _, plan := range plans {
		if len(plan) != 2 {
			continue
		}
		if plan[0].BlockerID == blk.ID() && plan[1].BlockerID == blk.ID() {
			return
		}
	}
	t.Fatalf("expected a plan where CanBlockAdditional blocker blocks both attackers; got %v", plans)
}

func TestAttackingPlayer_DeathtouchTrampleAssignsOnePerBlocker(t *testing.T) {
	g, pa, pb := makeGame()
	atk := makePerm("Venomous Wurm", "{4}{G}", 5, 5, pa.PlayerID(),
		mage.WithKeyword(core.Deathtouch), mage.WithKeyword(core.Trample))
	blk1 := makePerm("Wall One", "{2}{W}", 0, 3, pb.PlayerID())
	blk2 := makePerm("Wall Two", "{2}{W}", 0, 3, pb.PlayerID())
	g.AddToBattlefield(atk, blk1, blk2)

	ap := &attackingPlayer{}
	ordered := []*mage.Permanent{blk1, blk2}
	assignment := ap.GetCombatDamageAssignment(g, atk, ordered, atk.CurrentPower(g))
	if assignment[blk1.ID()] != 1 || assignment[blk2.ID()] != 1 {
		t.Fatalf("deathtouch trample assignment = %v, want 1 damage to each blocker", assignment)
	}
}
