package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// RunTurn executes a complete turn for the active player, reading steps
// from g.Schedule so that effects can skip or insert steps mid-turn.
// stopAt is checked: if we reach the specified turn+step, we stop.
func (g *Game) RunTurn(stopTurn int, stopStep PhaseStep) bool {
	if g.schedule == nil {
		g.schedule = newTurnSchedule()
	}
	// g.resetManaProducedThisTurn()
	g.schedule.buildNextTurn()
	for {
		step, ok := g.schedule.popNextStep()
		if !ok {
			// Put the stop step back so the next Run resumes correctly.
			g.schedule.Remaining = append([]PhaseStep{step}, g.schedule.Remaining...)
			return false
		}
		g.RunStepWithPriority(step)
		if g.stopped {
			return true
		}
	}
}

// RunStepWithPriority runs a single step of the turn using the priority system.
// It sets the step, applies continuous effects, performs step-specific actions,
// then runs a priority round (unless the step has no priority, e.g. Untap).
func (g *Game) RunStepWithPriority(step PhaseStep) {
	g.step = step

	// CR 500.5: any unspent mana empties as the step/phase ends.
	defer g.emptyManaPools()

	// Check SBAs after each step
	defer g.CheckStateBasedActions()

	g.effects.Apply(g)

	switch step {
	case Untap:
		g.doUntap()
		// no priority in untap

	case Upkeep:
		g.doUpkeepActions()
		g.runPriorityRound(false)

	case Draw:
		g.doDrawNormalDraw()
		g.doDrawActions()
		g.runPriorityRound(false)

	case PrecombatMain:
		g.runPriorityRound(true)

	case BeginCombat:
		g.doBeginCombatActions()
		g.runPriorityRound(false)

	case DeclareAttackers:
		g.doDeclareAttackers()
		if len(g.combat.Groups) > 0 {
			g.PutTriggersOnStack()
			g.runPriorityRound(false)
		}

	case DeclareBlockers:
		// 508.8: Skip if no creatures are attacking.
		if len(g.combat.Groups) > 0 {
			g.doDeclareBlockers()
			g.PutTriggersOnStack()
			g.runPriorityRound(false)
		}

	case FirstStrikeDamage:
		if g.combat.HasFirstStrikers(g) {
			g.resolvingCombatDamage = true
			g.combat.ResolveDamage(g, true)
			g.resolvingCombatDamage = false
			g.runPriorityRound(false)
		}

	case CombatDamage:
		if len(g.combat.Groups) > 0 {
			g.resolvingCombatDamage = true
			g.combat.ResolveDamage(g, false)
			g.resolvingCombatDamage = false
			g.runPriorityRound(false)
		}

	case EndCombat:
		g.FireEvent(GameEvent{
			Type:     EvtEndOfCombat,
			PlayerID: g.ActivePlayerObj().PlayerID(),
		})
		g.PutTriggersOnStack()
		g.runPriorityRound(false)
		g.effects.RemoveEndOfCombat()
		g.effects.Apply(g)
		g.combat.Reset()

	case PostcombatMain:
		g.runPriorityRound(true)

	case EndStep:
		g.doEndStepActions()
		g.runPriorityRound(false)

	case Cleanup:
		if g.doCleanupActions() {
			// CR 514.3a: triggers fired during cleanup — players get priority,
			// then another cleanup step begins.
			g.cleanupPriorityRounds++
			g.runPriorityRound(false)
			g.RunStepWithPriority(Cleanup)
		}
	}
}

func (g *Game) doBeginCombatActions() {
	active := g.ActivePlayerObj()
	g.FireEvent(GameEvent{
		Type:     EvtBeginCombat,
		PlayerID: active.PlayerID(),
	})
	g.PutTriggersOnStack()
}

func (g *Game) doEndStepActions() {
	active := g.ActivePlayerObj()
	g.FireEvent(GameEvent{
		Type:     EvtEndStep,
		PlayerID: active.PlayerID(),
	})
	g.PutTriggersOnStack()
}
