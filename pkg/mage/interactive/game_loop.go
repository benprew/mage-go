package interactive

import (
	"fmt"
	"time"

	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// RunGameLoop is the main interactive game loop running in a goroutine.
// RunGameLoop runs the game until it ends, communicating state to/from the TUI
// via channels embedded in the HumanPlayer at g.Players[humanIdx].
// aiActionPause is how long to pause after each visible AI action so the player
// can see what happened; 0 means no pause (useful for testing).
func RunGameLoop(g *mage.Game, humanIdx int, aiActionPause time.Duration) {
	hp := g.Players[humanIdx].(*HumanPlayer)
	defer close(hp.toTUI)
	defer close(hp.choiceReqs)

	aiIdx := (humanIdx + 1) % 2
	aiPlayer, isAI := g.Players[aiIdx].(*AIPlayer)
	_ = aiPlayer

	var gameLog []string
	var lastUndo undoSnapshot

	addLog := func(msg string) {
		gameLog = append(gameLog, msg)
		if len(gameLog) > 100 {
			gameLog = gameLog[len(gameLog)-100:]
		}
		hp.gameLog = gameLog
	}

	send := func(prompt PromptType, options []ActionOption) {
		state := SnapshotGameState(g, humanIdx)
		hp.toTUI <- GameMsg{
			State:    state,
			Prompt:   prompt,
			Options:  options,
			Log:      append([]string{}, gameLog...),
			GameOver: g.IsGameOver(),
			Winner:   g.Winner(),
			CanUndo:  lastUndo.valid,
		}
	}

	getHumanAction := func(mainPhase bool) PriorityAction {
		playerID := g.Players[humanIdx].PlayerID()
		options := GetAvailableActions(g, playerID, g.LandsPlayedThisTurn, mainPhase)
		if len(options) == 1 && options[0].Type == ActionPass && !lastUndo.valid {
			return PriorityAction{Type: ActionPass}
		}
		if mainPhase {
			send(PromptMainPhaseAction, options)
		} else {
			send(PromptPriority, options)
		}
		return <-hp.fromTUI
	}

	getAIAction := func(mainPhase bool) PriorityAction {
		if !isAI {
			return PriorityAction{Type: ActionPass}
		}
		return aiPlayer.GetPriorityAction(g, g.LandsPlayedThisTurn, mainPhase)
	}

	// showAIAction sends an updated game state snapshot and briefly pauses so the
	// player can see the result of each AI action before the game moves on.
	showAIAction := func() {
		send(PromptNone, nil)
		if aiActionPause > 0 {
			time.Sleep(aiActionPause)
		}
	}

	getAction := func(playerIdx int, mainPhase bool) PriorityAction {
		if playerIdx == humanIdx {
			return getHumanAction(mainPhase)
		}
		return getAIAction(mainPhase)
	}

	runPriorityLoop := func(mainPhase bool) {
		for {
			if g.IsGameOver() {
				return
			}

			activeIdx := g.ActivePlayer
			nonActiveIdx := (g.ActivePlayer + 1) % 2

			action := getAction(activeIdx, mainPhase)
			if action.Type == ActionUndo && lastUndo.valid {
				restoreFromUndo(g, g.Players[humanIdx].PlayerID(), lastUndo)
				if len(gameLog) > lastUndo.logLen {
					gameLog = gameLog[:lastUndo.logLen]
				}
				lastUndo.valid = false
				addLog("Undid last action")
				continue
			}
			if action.Type != ActionPass {
				if activeIdx == humanIdx {
					lastUndo = captureForUndo(g, g.Players[humanIdx].PlayerID(), len(gameLog))
				} else {
					lastUndo.valid = false
				}
				executeAction(g, g.Players[activeIdx].PlayerID(), action, addLog)
				g.CheckStateBasedActions()
				if g.IsGameOver() {
					return
				}
				if activeIdx != humanIdx {
					showAIAction()
				}
				continue
			}

			action = getAction(nonActiveIdx, false)
			if action.Type == ActionUndo && lastUndo.valid {
				restoreFromUndo(g, g.Players[humanIdx].PlayerID(), lastUndo)
				if len(gameLog) > lastUndo.logLen {
					gameLog = gameLog[:lastUndo.logLen]
				}
				lastUndo.valid = false
				addLog("Undid last action")
				continue
			}
			if action.Type != ActionPass {
				if nonActiveIdx == humanIdx {
					lastUndo = captureForUndo(g, g.Players[humanIdx].PlayerID(), len(gameLog))
				} else {
					lastUndo.valid = false
				}
				executeAction(g, g.Players[nonActiveIdx].PlayerID(), action, addLog)
				g.CheckStateBasedActions()
				if g.IsGameOver() {
					return
				}
				if nonActiveIdx != humanIdx {
					showAIAction()
				}
				continue
			}

			lastUndo.valid = false
			if g.Stack.IsEmpty() {
				return
			}

			top := g.Stack.Peek()
			topName := "ability"
			if top.Card != nil {
				topName = top.Card.Name()
			}
			addLog(fmt.Sprintf("Resolving %s%s", topName, targetSuffix(g, top.Targets)))
			g.ResolveTopOfStack()
			g.CheckStateBasedActions()
			if g.IsGameOver() {
				return
			}
		}
	}

	for g.Turn <= 100 {
		for _, step := range core.AllSteps() {
			g.Step = step
			g.Effects.Apply(g)

			if g.IsGameOver() {
				send(PromptNone, nil)
				return
			}

			switch step {
			case core.Untap:
				g.DoUntap()
				addLog(fmt.Sprintf("── Turn %d: %s ──", g.Turn, g.ActivePlayerObj().Name()))
				if g.ActivePlayer != humanIdx {
					send(PromptNone, nil) // show state at start of AI's turn
				}

			case core.Upkeep:
				g.DoUpkeep()

			case core.Draw:
				g.DoDraw()
				if g.ActivePlayer == humanIdx {
					addLog("You draw a card")
				} else {
					addLog(fmt.Sprintf("%s draws a card", g.ActivePlayerObj().Name()))
				}

			case core.PrecombatMain:
				runPriorityLoop(true)

			case core.BeginCombat:
				// nothing

			case core.DeclareAttackers:
				activeIdx := g.ActivePlayer
				attackerIDs := g.Players[activeIdx].DeclareAttackers(g)
				if len(attackerIDs) > 0 {
					performAttack(g, attackerIDs, addLog)
					if activeIdx != humanIdx {
						showAIAction()
					}
				} else if activeIdx == humanIdx {
					addLog("You choose not to attack")
				}
				if len(g.Combat.Groups) > 0 {
					runPriorityLoop(false)
				}

			case core.DeclareBlockers:
				if len(g.Combat.Groups) == 0 {
					continue
				}
				nonActiveIdx := (g.ActivePlayer + 1) % 2
				blockers := g.Players[nonActiveIdx].DeclareBlockers(g)
				if len(blockers) > 0 {
					performBlock(g, blockers, addLog)
					if nonActiveIdx != humanIdx {
						showAIAction()
					}
				}
				if len(g.Combat.Groups) > 0 {
					runPriorityLoop(false)
				}

			case core.FirstStrikeDamage:
				if g.Combat.HasFirstStrikers(g) {
					g.Combat.ResolveDamage(g, true)
					g.CheckStateBasedActions()
				}

			case core.CombatDamage:
				if len(g.Combat.Groups) > 0 {
					addLog("── Combat damage ──")
					logCombatPreview(g, addLog)
					g.Combat.ResolveDamage(g, false)
					g.CheckStateBasedActions()
					reportCombatResults(g, humanIdx, addLog)
					showAIAction() // always show damage result
				}

			case core.EndCombat:
				g.Combat.Reset()

			case core.PostcombatMain:
				runPriorityLoop(true)

			case core.EndStep:
				g.FireEvent(core.GameEvent{
					Type:     core.EvtEndStep,
					PlayerID: g.ActivePlayerObj().PlayerID(),
				})
				g.PutTriggersOnStack()
				if !g.Stack.IsEmpty() {
					runPriorityLoop(false)
				}

			case core.Cleanup:
				g.DoCleanup()
			}

			g.CheckStateBasedActions()
			if g.IsGameOver() {
				send(PromptNone, nil)
				return
			}
		}

		if len(g.ExtraTurns) > 0 {
			extraPlayerID := g.ExtraTurns[0]
			g.ExtraTurns = g.ExtraTurns[1:]
			for i, p := range g.Players {
				if p.PlayerID() == extraPlayerID {
					g.ActivePlayer = i
					break
				}
			}
		} else {
			g.ActivePlayer = (g.ActivePlayer + 1) % len(g.Players)
		}
		g.Turn++
	}
}
