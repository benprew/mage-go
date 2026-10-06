package mage

import . "github.com/benprew/mage-go/pkg/mage/core"

type combatTimingHistory struct {
	firstProgress   PhaseStep
	currentProgress PhaseStep
	firstEnded      bool
}

func (h *combatTimingHistory) enter(previous, next PhaseStep) {
	if previous == EndCombat && next != EndCombat {
		h.closeWindows()
		h.firstEnded = true
	}
	if next >= BeginCombat && next <= EndCombat && (previous < BeginCombat || previous > EndCombat || (previous == EndCombat && next < EndCombat) || (next == BeginCombat && h.currentProgress == EndCombat)) {
		h.currentProgress = next
	}
	if next >= BeginCombat && next <= EndCombat {
		h.currentProgress = max(h.currentProgress, next)
		if !h.firstEnded {
			h.firstProgress = max(h.firstProgress, next)
		}
	} else if next >= PostcombatMain || (previous >= BeginCombat && previous <= EndCombat) {
		h.closeWindows()
		h.firstEnded = true
	}
}

func (h *combatTimingHistory) closeWindows() {
	h.currentProgress = EndCombat
	h.firstProgress = EndCombat
}

type beforeCombatStepCondition struct {
	point  PhaseStep
	during bool
}

// BeforeCombatStep uses combat history because later combats must not restore the first window.
func BeforeCombatStep(point PhaseStep, during bool) StateCondition {
	return beforeCombatStepCondition{point, during}
}
func (beforeCombatStepCondition) stateCondition() {}
func (c beforeCombatStepCondition) checkState(g GameReader, _ StateContext) bool {
	return g.CombatWindowOpen(c.point, c.during)
}
func (c beforeCombatStepCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", Text: "before " + c.point.String()}
	if c.point != DeclareAttackers && c.point != DeclareBlockers && c.point != CombatDamage {
		n.problem = "supply attackers, blockers, or combat damage"
	}
	return n
}

// CombatWindowOpen reads the first combat or the current combat without changing state.
func (g *Game) CombatWindowOpen(point PhaseStep, during bool) bool {
	if point != DeclareAttackers && point != DeclareBlockers && point != CombatDamage {
		return false
	}
	if point == CombatDamage {
		point = FirstStrikeDamage
	}
	h := g.turns.combatTiming
	if during {
		return g.GetStep() >= BeginCombat && g.GetStep() <= EndCombat && h.currentProgress < point
	}
	return !h.firstEnded && h.firstProgress < point
}
