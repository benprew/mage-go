package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// PriorityActionType identifies what kind of action a player takes when
// they receive priority.
type PriorityActionType int

const (
	PriorityPass PriorityActionType = iota
	PriorityCastSpell
	PriorityActivateAbility
	PriorityPlayLand
)

// PriorityAction is what a player wants to do when they have priority.
type PriorityAction struct {
	Type        PriorityActionType
	CardID      uuid.UUID
	Targets     []uuid.UUID
	PermanentID uuid.UUID
	AbilityIdx  int
	XValue      int
}

// PriorityHandler is called by the engine whenever a player receives priority.
// It returns the action the player wants to take.
//   - g is the current game state
//   - playerIdx is the index into g.Players
//   - mainPhase is true during PrecombatMain and PostcombatMain
type PriorityHandler func(g *Game, playerIdx int, mainPhase bool) PriorityAction

// executePriorityAction executes a non-pass priority action for the given player.
func (g *Game) executePriorityAction(playerIdx int, action PriorityAction) {
	playerID := g.Players[playerIdx].PlayerID()
	switch action.Type {
	case PriorityPlayLand:
		_ = g.playLandCore(playerID, action.CardID)
	case PriorityCastSpell:
		_ = g.CastSpellByID(playerID, action.CardID, action.Targets, action.XValue)
	case PriorityActivateAbility:
		_ = g.ActivateAbilityByIndex(playerID, action.PermanentID, action.AbilityIdx, action.Targets)
	}
}

// RunPriorityRound runs the full priority loop for the current step:
//
//	SBA check → trigger placement → cycle players → resolve one at a time.
//
// If OnPriority is nil, falls back to draining the stack atomically (ResolveStack).
func (g *Game) RunPriorityRound(mainPhase bool) {
	if g.OnPriority == nil {
		g.CheckStateBasedActions()
		g.ResolveStack()
		return
	}

	for {
		// 1. Check state-based actions (includes lethal damage, 0-toughness, etc.)
		g.CheckStateBasedActions()

		// 2. Put pending triggers on stack
		g.PutTriggersOnStack()

		// 3. Cycle through players starting from active player
		allPassed := true
		for i := 0; i < len(g.Players); i++ {
			playerIdx := (g.ActivePlayer + i) % len(g.Players)
			action := g.OnPriority(g, playerIdx, mainPhase)
			if action.Type != PriorityPass {
				g.executePriorityAction(playerIdx, action)
				if g.AfterPriorityAction != nil {
					g.AfterPriorityAction(g, playerIdx, action)
				}
				allPassed = false
				break // restart loop from SBA check
			}
		}

		if !allPassed {
			continue
		}

		// All players passed in succession
		if g.Stack.IsEmpty() {
			return // step proceeds
		}

		// Resolve top of stack, then restart priority
		if g.BeforeStackResolve != nil {
			g.BeforeStackResolve(g)
		}
		g.ResolveTopOfStack()
	}
}

func countBattlefield(g *Game, playerID uuid.UUID) int {
	count := 0
	for _, p := range g.Battlefield {
		if p.Controller == playerID {
			count++
		}
	}
	return count
}

// RunStepWithPriority runs a single step of the turn using the priority system.
// It sets the step, applies continuous effects, performs step-specific actions,
// then runs a priority round (unless the step has no priority, e.g. Untap).
func (g *Game) RunStepWithPriority(step PhaseStep) {
	g.Step = step
	g.Effects.Apply(g)

	switch step {
	case Untap:
		g.doUntap()
		return // no priority in untap

	case Upkeep:
		g.doUpkeepActions()
		g.RunPriorityRound(false)

	case Draw:
		g.doDrawActions()
		g.RunPriorityRound(false)
		g.doDrawNormalDraw()
		g.RunPriorityRound(false)

	case PrecombatMain:
		g.RunPriorityRound(true)

	case BeginCombat:
		g.doBeginCombatActions()
		g.RunPriorityRound(false)

	case DeclareAttackers:
		g.doDeclareAttackers()
		if len(g.Combat.Groups) > 0 {
			g.PutTriggersOnStack()
			g.RunPriorityRound(false)
		}

	case DeclareBlockers:
		// 508.8: Skip if no creatures are attacking.
		if len(g.Combat.Groups) > 0 {
			g.doDeclareBlockers()
			g.PutTriggersOnStack()
			g.RunPriorityRound(false)
		}

	case FirstStrikeDamage:
		if g.Combat.HasFirstStrikers(g) {
			g.resolvingCombatDamage = true
			g.Combat.ResolveDamage(g, true)
			g.resolvingCombatDamage = false
			g.RunPriorityRound(false)
		}

	case CombatDamage:
		if len(g.Combat.Groups) > 0 {
			g.resolvingCombatDamage = true
			g.Combat.ResolveDamage(g, false)
			g.resolvingCombatDamage = false
			g.RunPriorityRound(false)
		}

	case EndCombat:
		g.FireEvent(GameEvent{
			Type:     EvtEndOfCombat,
			PlayerID: g.ActivePlayerObj().PlayerID(),
		})
		g.PutTriggersOnStack()
		g.RunPriorityRound(false)
		g.Effects.RemoveEndOfCombat()
		g.Effects.Apply(g)
		g.Combat.Reset()

	case PostcombatMain:
		g.RunPriorityRound(true)

	case EndStep:
		g.doEndStepActions()
		g.RunPriorityRound(false)

	case Cleanup:
		if g.doCleanupActions() {
			// Triggers fired during cleanup — players get priority, then repeat cleanup.
			g.RunPriorityRound(false)
			g.CheckStateBasedActions()
			g.RunStepWithPriority(Cleanup)
			return
		}
	}

	// Check SBAs after each step (matches RunStep behavior)
	g.CheckStateBasedActions()
}
