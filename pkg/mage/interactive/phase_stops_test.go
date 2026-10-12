package interactive

import (
	"fmt"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPhaseStopsPriority(t *testing.T) {
	hp := NewHumanPlayer("You")
	g := mage.NewGame(hp, mage.NewBasePlayer("Opponent"))
	hp.EnablePhaseStops()
	for _, tc := range []struct {
		step      core.PhaseStep
		own, want bool
	}{
		{core.PrecombatMain, true, false}, {core.PostcombatMain, true, false},
		{core.Upkeep, true, true}, {core.EndStep, false, false},
		{core.PrecombatMain, false, true}, {core.BeginCombat, true, true},
	} {
		g.SetStep(tc.step)
		if !tc.own {
			g.SetActivePlayerIndex(1)
		} else {
			g.SetActivePlayerIndex(0)
		}
		if got := hp.autoPassPriority(g, []ActionOption{{Type: ActionPass}}); got != tc.want {
			t.Errorf("step %v own %v: got %v want %v", tc.step, tc.own, got, tc.want)
		}
	}
	hp.SetPhaseStop(core.Upkeep, true, true)
	g.SetStep(core.Upkeep)
	g.SetActivePlayerIndex(0)
	if hp.autoPassPriority(g, nil) {
		t.Fatal("explicit stop must give priority")
	}
	hp.SetPhaseStop(core.Upkeep, true, false)
	if !hp.autoPassPriority(g, nil) {
		t.Fatal("removed stop must pass")
	}
}

func TestPhaseStopsStackAndAttackers(t *testing.T) {
	hp := NewHumanPlayer("You")
	g := mage.NewGame(hp, mage.NewBasePlayer("Opponent"))
	hp.EnablePhaseStops()
	actions := []ActionOption{{Type: ActionActivateAbility}, {Type: ActionPass}}
	g.SetStep(core.Upkeep)
	for _, active := range []int{0, 1} {
		g.SetActivePlayerIndex(active)
		if !hp.autoPassPriority(g, actions) {
			t.Fatal("empty stack must pass even with available actions")
		}
		g.GetStack().Push(&mage.StackObject{})
		if hp.autoPassPriority(g, []ActionOption{{Type: ActionPass}}) {
			t.Fatal("stack must give priority even with no actions")
		}
		g.GetStack().Pop()
	}
	g.SetActivePlayerIndex(0)
	g.SetStep(core.BeginCombat)
	creature := makePerm("Bear", "{1}{G}", 2, 2, hp.PlayerID())
	g.AddToBattlefield(creature)
	if hp.autoPassPriority(g, actions) {
		t.Fatal("eligible attacker must stop at combat")
	}
	creature.Tapped = true
	if !hp.autoPassPriority(g, actions) {
		t.Fatal("tapped creature must not stop at combat")
	}
	creature.Tapped = false
	creature.GrantBaseAttr(core.AttrSummonSick)
	if !hp.autoPassPriority(g, actions) {
		t.Fatal("summoning sick creature must not stop at combat")
	}
}

func TestPhaseStopsOpponentAttack(t *testing.T) {
	hp := NewHumanPlayer("You")
	pb := mage.NewBasePlayer("Opponent")
	g := mage.NewGame(hp, pb)
	hp.EnablePhaseStops()
	g.SetActivePlayerIndex(1)
	g.SetStep(core.DeclareAttackers)
	if !hp.autoPassPriority(g, nil) {
		t.Fatal("no attack must pass")
	}
	attacker := makePerm("Bear", "{1}{G}", 2, 2, pb.PlayerID())
	g.AddToBattlefield(attacker)
	g.GetCombat().AddAttacker(attacker.ID(), hp.PlayerID())
	if hp.autoPassPriority(g, nil) {
		t.Fatal("declared attack must give priority")
	}
}

func TestPhaseStopsActiveCombatPriority(t *testing.T) {
	for _, ownTurn := range []bool{true, false} {
		for _, step := range []core.PhaseStep{core.DeclareAttackers, core.DeclareBlockers, core.FirstStrikeDamage, core.CombatDamage, core.EndCombat} {
			t.Run(fmt.Sprintf("own_%t_step_%v", ownTurn, step), func(t *testing.T) {
				hp := NewHumanPlayer("You")
				pb := mage.NewBasePlayer("Opponent")
				g := mage.NewGame(hp, pb)
				hp.EnablePhaseStops()
				var attacking, defending mage.Player = hp, pb
				if !ownTurn {
					g.SetActivePlayerIndex(1)
					attacking, defending = pb, hp
				}
				g.SetStep(step)
				attacker := makePerm("Bear", "{1}{G}", 2, 2, attacking.PlayerID())
				g.AddToBattlefield(attacker)
				g.GetCombat().AddAttacker(attacker.ID(), defending.PlayerID())
				for _, actions := range [][]ActionOption{
					{{Type: ActionPass}},
					{{Type: ActionCastSpell}, {Type: ActionPass}},
				} {
					if hp.autoPassPriority(g, actions) {
						t.Fatal("active combat must give human priority")
					}
				}
			})
		}
	}
}
