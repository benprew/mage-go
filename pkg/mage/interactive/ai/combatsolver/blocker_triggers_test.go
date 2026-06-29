package combatsolver

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"

	_ "github.com/benprew/mage-go/cards/fourthedition"
	_ "github.com/benprew/mage-go/cards/legends"
)

func realPerm(t *testing.T, name string, owner uuid.UUID) *mage.Permanent {
	t.Helper()
	card, err := mage.CreateCard(name)
	if err != nil {
		t.Fatalf("CreateCard(%q): %v", name, err)
	}
	card.SetOwner(owner)
	perm := mage.NewPermanent(card, owner)
	perm.RevokeBaseAttr(core.AttrSummonSick)
	// Mirror PutOnBattlefield: real permanents enter the battlefield with their
	// ability sources/controllers set, which trigger conditions depend on.
	for _, a := range perm.RuntimeAbilities {
		a.SetSource(perm.ID())
		a.SetController(owner)
	}
	return perm
}

// The combat search clone must simulate declare-blockers triggers. Murk
// Dwellers' "Whenever ~ attacks and isn't blocked, it gets +2/+0" fires on
// EvtBlockersDecl; if ExecuteBlockers doesn't fire that event, the solver
// evaluates an unblocked Murk Dwellers as a 2/2 dealing 2 instead of a 4/2
// dealing 4 — undervaluing it and mis-modeling combat.
func TestSolverSimulatesUnblockedAttackTrigger(t *testing.T) {
	g, pa, pb := makeGame()
	murk := realPerm(t, "Murk Dwellers", pa.PlayerID())
	g.AddToBattlefield(murk)
	pb.SetLife(20)

	g.ExecuteAttackers(pa.PlayerID(), []uuid.UUID{murk.ID()})

	// Replay the no-block branch the way SolveDefense does.
	clone := g.Clone()
	clone.ExecuteBlockers(nil)
	clone.ExecuteCombatDamage()
	clone.CheckStateBasedActions()

	if got := clone.GetPlayer(pb.PlayerID()).Life(); got != 16 {
		t.Errorf("unblocked Murk Dwellers should deal 4 (life 16), got life %d (+2/+0 trigger not simulated)", got)
	}
}

// A blocked Murk Dwellers must NOT get its +2/+0, even when blocked by
// Abomination (which also triggers on EvtBlockersDecl).
func TestSolverBlockedAttackerNoBoost(t *testing.T) {
	g, pa, pb := makeGame()
	murk := realPerm(t, "Murk Dwellers", pa.PlayerID())
	abom := realPerm(t, "Abomination", pb.PlayerID())
	g.AddToBattlefield(murk, abom)

	g.ExecuteAttackers(pa.PlayerID(), []uuid.UUID{murk.ID()})
	clone := g.Clone()
	clone.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: abom.ID(), AttackerID: murk.ID()}})

	if got := clone.FindPermanent(murk.ID()).CurrentPower(clone); got != 2 {
		t.Errorf("blocked Murk Dwellers should stay 2 power, got %d", got)
	}
}
