package mage

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Functions related to running a priority round (aka the stack)

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
//   - playerIdx is the index into g.players
//   - mainPhase is true during PrecombatMain and PostcombatMain
type PriorityHandler func(g *Game, playerIdx int, mainPhase bool) PriorityAction

// executePriorityAction executes a non-pass priority action for the given player.
// Returns true if the action succeeded, false if it failed.
func (g *Game) executePriorityAction(playerIdx int, action PriorityAction) bool {
	playerID := g.players[playerIdx].PlayerID()
	switch action.Type {
	case PriorityPlayLand:
		if err := g.playLandCore(playerID, action.CardID); err != nil {
			if DebugPriority {
				fmt.Printf("[PRIORITY] PlayLand FAILED player=%s cardID=%s err=%v\n",
					g.players[playerIdx].Name(), action.CardID, err)
			}
			return false
		}
	case PriorityCastSpell:
		if err := g.CastSpellByID(playerID, action.CardID, action.Targets, action.XValue); err != nil {
			if DebugPriority {
				fmt.Printf("[PRIORITY] CastSpell FAILED player=%s cardID=%s targets=%v err=%v\n",
					g.players[playerIdx].Name(), action.CardID, action.Targets, err)
			}
			return false
		}
	case PriorityActivateAbility:
		prevX := g.currentX
		if action.XValue > 0 {
			g.currentX = action.XValue
		}
		err := g.ActivateAbilityByIndex(playerID, action.PermanentID, action.AbilityIdx, action.Targets)
		g.currentX = prevX
		if err != nil {
			if DebugPriority {
				fmt.Printf("[PRIORITY] ActivateAbility FAILED player=%s permID=%s abilityIdx=%d targets=%v err=%v\n",
					g.players[playerIdx].Name(), action.PermanentID, action.AbilityIdx, action.Targets, err)
			}
			return false
		}
	}
	return true
}

// runPriorityRound runs the full priority loop for the current step:
//
//	SBA check → trigger placement → cycle players → resolve one at a time.
//
// If OnPriority is nil, falls back to draining the stack atomically (ResolveStack).
func (g *Game) runPriorityRound(mainPhase bool) {
	if g.onPriority == nil {
		g.CheckStateBasedActions()
		g.ResolveStack()
		return
	}
	if EngineTimingEnabled() {
		addEnginePriorityRound()
	}

	iterations := 0
	for {
		iterations++
		timingEnabled := EngineTimingEnabled()
		if timingEnabled {
			addEnginePriorityIteration()
		}
		if DebugPriority && iterations%50 == 0 {
			fmt.Printf("[PRIORITY] WARNING: %d iterations in RunPriorityRound turn=%d step=%s mainPhase=%v stackSize=%d\n",
				iterations, g.turn, g.step, mainPhase, g.stack.Size())
			for i, p := range g.players {
				fmt.Printf("[PRIORITY]   player[%d]=%s life=%d hand=%d battlefield=%d\n",
					i, p.Name(), p.Life(), len(p.Hand()), countBattlefield(g, p.PlayerID()))
			}
		}
		if iterations > 500 {
			fmt.Printf("[PRIORITY] EMERGENCY: breaking out of priority loop after %d iterations turn=%d step=%s\n",
				iterations, g.turn, g.step)
			return
		}

		// 1. Check state-based actions (includes lethal damage, 0-toughness, etc.)
		phaseStart := time.Time{}
		if timingEnabled {
			phaseStart = time.Now()
		}
		g.CheckStateBasedActions()
		if timingEnabled {
			addEngineTiming(&engineTimingSBANs, time.Since(phaseStart))
			phaseStart = time.Now()
		}

		// 2. Evaluate state triggers (CR 603.8) — they're checked alongside SBA.
		g.CheckStateTriggers()

		// 3. Put pending triggers on stack
		g.PutTriggersOnStack()
		if timingEnabled {
			addEngineTiming(&engineTimingTriggersNs, time.Since(phaseStart))
		}

		// 3. Cycle through players starting from active player
		allPassed := true
		for i := 0; i < len(g.players); i++ {
			playerIdx := (g.activePlayer + i) % len(g.players)
			if timingEnabled {
				phaseStart = time.Now()
			}
			action := g.onPriority(g, playerIdx, mainPhase)
			if timingEnabled {
				addEngineTiming(&engineTimingOnPriorityNs, time.Since(phaseStart))
			}
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
						for _, c := range g.players[playerIdx].Hand() {
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
						g.players[playerIdx].Name(), actionName, cardName, action.Targets, iterations)
				}
				if timingEnabled {
					phaseStart = time.Now()
				}
				executed := g.executePriorityAction(playerIdx, action)
				if timingEnabled {
					addEngineTiming(&engineTimingExecuteNs, time.Since(phaseStart))
					addEngineExecutedAction()
				}
				if executed {
					if g.afterPriorityAction != nil {
						if timingEnabled {
							phaseStart = time.Now()
						}
						g.afterPriorityAction(g, playerIdx, action)
						if timingEnabled {
							addEngineTiming(&engineTimingAfterNs, time.Since(phaseStart))
						}
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
		if g.stack.IsEmpty() {
			return // step proceeds
		}

		// Resolve top of stack, then restart priority
		if g.beforeStackResolve != nil {
			g.beforeStackResolve(g)
		}
		if timingEnabled {
			phaseStart = time.Now()
		}
		g.ResolveTopOfStack()
		if timingEnabled {
			addEngineTiming(&engineTimingResolveNs, time.Since(phaseStart))
		}
	}
}
