package mage

import (
	"fmt"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// DebugPriority enables verbose logging of priority actions and errors.
var DebugPriority bool

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
// Returns true if the action succeeded, false if it failed.
func (g *Game) executePriorityAction(playerIdx int, action PriorityAction) bool {
	playerID := g.Players[playerIdx].PlayerID()
	switch action.Type {
	case PriorityPlayLand:
		if err := g.playLandCore(playerID, action.CardID); err != nil {
			if DebugPriority {
				fmt.Printf("[PRIORITY] PlayLand FAILED player=%s cardID=%s err=%v\n",
					g.Players[playerIdx].Name(), action.CardID, err)
			}
			return false
		}
	case PriorityCastSpell:
		if err := g.CastSpellByID(playerID, action.CardID, action.Targets, action.XValue); err != nil {
			if DebugPriority {
				fmt.Printf("[PRIORITY] CastSpell FAILED player=%s cardID=%s targets=%v err=%v\n",
					g.Players[playerIdx].Name(), action.CardID, action.Targets, err)
			}
			return false
		}
	case PriorityActivateAbility:
		if err := g.ActivateAbilityByIndex(playerID, action.PermanentID, action.AbilityIdx, action.Targets); err != nil {
			if DebugPriority {
				fmt.Printf("[PRIORITY] ActivateAbility FAILED player=%s permID=%s abilityIdx=%d targets=%v err=%v\n",
					g.Players[playerIdx].Name(), action.PermanentID, action.AbilityIdx, action.Targets, err)
			}
			return false
		}
	}
	return true
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

	iterations := 0
	for {
		iterations++
		if DebugPriority && iterations%50 == 0 {
			fmt.Printf("[PRIORITY] WARNING: %d iterations in RunPriorityRound turn=%d step=%s mainPhase=%v stackSize=%d\n",
				iterations, g.Turn, g.Step, mainPhase, g.Stack.Size())
			for i, p := range g.Players {
				fmt.Printf("[PRIORITY]   player[%d]=%s life=%d hand=%d battlefield=%d\n",
					i, p.Name(), p.Life(), len(p.Hand()), countBattlefield(g, p.PlayerID()))
			}
		}
		if iterations > 500 {
			fmt.Printf("[PRIORITY] EMERGENCY: breaking out of priority loop after %d iterations turn=%d step=%s\n",
				iterations, g.Turn, g.Step)
			return
		}

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
				if DebugPriority {
					actionName := "unknown"
					cardName := ""
					switch action.Type {
					case PriorityPlayLand:
						actionName = "PlayLand"
					case PriorityCastSpell:
						actionName = "CastSpell"
					case PriorityActivateAbility:
						actionName = "ActivateAbility"
					}
					if action.CardID != uuid.Nil {
						for _, c := range g.Players[playerIdx].Hand() {
							if c.ID() == action.CardID {
								cardName = c.Name()
								break
							}
						}
					}
					if action.PermanentID != uuid.Nil {
						if perm := g.FindPermanent(action.PermanentID); perm != nil {
							cardName = perm.Name()
						}
					}
					fmt.Printf("[PRIORITY] player=%s action=%s card=%q targets=%v iter=%d\n",
						g.Players[playerIdx].Name(), actionName, cardName, action.Targets, iterations)
				}
				if g.executePriorityAction(playerIdx, action) {
					if g.AfterPriorityAction != nil {
						g.AfterPriorityAction(g, playerIdx, action)
					}
					allPassed = false
					break // restart loop from SBA check
				}
				// Action failed — treat as pass for this player
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
	g.inStep = true
	defer func() { g.inStep = false }()
	// CR 500.5: empty mana pools as the step/phase ends.
	defer g.EmptyManaPools()
	g.Effects.Apply(g)

	switch step {
	case Untap:
		g.doUntap()
		return // no priority in untap

	case Upkeep:
		g.doUpkeepActions()
		g.RunPriorityRound(false)

	case Draw:
		g.doDrawNormalDraw()
		g.doDrawActions()
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
			// CR 514.3a: triggers fired during cleanup — players get priority,
			// then another cleanup step begins.
			g.CleanupPriorityRounds++
			g.RunPriorityRound(false)
			g.CheckStateBasedActions()
			g.RunStepWithPriority(Cleanup)
			return
		}
	}

	// Check SBAs after each step (matches RunStep behavior)
	g.CheckStateBasedActions()
}
