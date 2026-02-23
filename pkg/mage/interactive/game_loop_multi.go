package interactive

import (
	"fmt"

	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// RunMultiplayerGameLoop runs a two-human-player game, communicating with each
// player's TUI independently via their PlayerChannels. It mirrors RunGameLoop
// but has no AI logic and no undo system.
func RunMultiplayerGameLoop(g *mage.Game, channels [2]PlayerChannels) {
	defer close(channels[0].ToPlayer)
	defer close(channels[1].ToPlayer)
	// Also close each player's choice request channel so their TUI can exit cleanly.
	if hp, ok := g.Players[0].(*HumanPlayer); ok {
		defer close(hp.choiceReqs)
	}
	if hp, ok := g.Players[1].(*HumanPlayer); ok {
		defer close(hp.choiceReqs)
	}

	var gameLog []string

	addLog := func(msg string) {
		gameLog = append(gameLog, msg)
		if len(gameLog) > 100 {
			gameLog = gameLog[len(gameLog)-100:]
		}
	}

	// sendTo sends a game state snapshot from idx's perspective to that player.
	sendTo := func(idx int, prompt PromptType, options []ActionOption) {
		state := SnapshotGameState(g, idx)
		msg := GameMsg{
			State:    state,
			Prompt:   prompt,
			Options:  options,
			Log:      append([]string{}, gameLog...),
			GameOver: g.IsGameOver(),
			Winner:   g.Winner(),
		}
		defer func() { recover() }() // guard against send on closed channel
		channels[idx].ToPlayer <- msg
	}

	// broadcast sends PromptNone to both players so they see the current board.
	broadcast := func() {
		sendTo(0, PromptNone, nil)
		sendTo(1, PromptNone, nil)
	}

	// readFrom reads a PriorityAction from player idx, returning ok=false on disconnect.
	readFrom := func(idx int) (PriorityAction, bool) {
		action, ok := <-channels[idx].FromPlayer
		return action, ok
	}

	// handleDisconnect sends a game-over message to the surviving player.
	handleDisconnect := func(disconnectedIdx int) {
		survivorIdx := (disconnectedIdx + 1) % 2
		survivor := g.Players[survivorIdx]
		func() {
			defer func() { recover() }()
			channels[survivorIdx].ToPlayer <- GameMsg{
				GameOver: true,
				Winner:   survivor.Name(),
				Log:      append([]string{}, gameLog...),
			}
		}()
	}

	getAction := func(idx int, mainPhase bool) (PriorityAction, bool) {
		playerID := g.Players[idx].PlayerID()
		options := GetAvailableActions(g, playerID, g.LandsPlayedThisTurn, mainPhase)
		if len(options) == 1 && options[0].Type == ActionPass {
			return PriorityAction{Type: ActionPass}, true
		}
		if mainPhase {
			sendTo(idx, PromptMainPhaseAction, options)
		} else {
			sendTo(idx, PromptPriority, options)
		}
		return readFrom(idx)
	}

	runPriorityLoop := func(mainPhase bool) bool {
		for {
			if g.IsGameOver() {
				return true
			}

			activeIdx := g.ActivePlayer
			nonActiveIdx := (g.ActivePlayer + 1) % 2

			action, ok := getAction(activeIdx, mainPhase)
			if !ok {
				handleDisconnect(activeIdx)
				return false
			}
			if action.Type != ActionPass {
				executeAction(g, g.Players[activeIdx].PlayerID(), action, addLog)
				g.CheckStateBasedActions()
				if g.IsGameOver() {
					return true
				}
				broadcast()
				continue
			}

			action, ok = getAction(nonActiveIdx, false)
			if !ok {
				handleDisconnect(nonActiveIdx)
				return false
			}
			if action.Type != ActionPass {
				executeAction(g, g.Players[nonActiveIdx].PlayerID(), action, addLog)
				g.CheckStateBasedActions()
				if g.IsGameOver() {
					return true
				}
				broadcast()
				continue
			}

			if g.Stack.IsEmpty() {
				return true
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
				return true
			}
		}
	}

	for g.Turn <= 100 {
		for _, step := range core.AllSteps() {
			g.Step = step
			g.Effects.Apply(g)

			if g.IsGameOver() {
				broadcast()
				return
			}

			switch step {
			case core.Untap:
				g.DoUntap()
				addLog(fmt.Sprintf("── Turn %d: %s ──", g.Turn, g.ActivePlayerObj().Name()))
				broadcast()

			case core.Upkeep:
				g.DoUpkeep()

			case core.Draw:
				g.DoDraw()
				addLog(fmt.Sprintf("%s draws a card", g.ActivePlayerObj().Name()))

			case core.PrecombatMain:
				if !runPriorityLoop(true) {
					return
				}

			case core.BeginCombat:
				// nothing

			case core.DeclareAttackers:
				activeIdx := g.ActivePlayer
				eligible := getEligibleAttackers(g, g.Players[activeIdx].PlayerID())
				if len(eligible) > 0 {
					sendTo(activeIdx, PromptDeclareAttackers, attackerOptions(eligible))
					sendTo((activeIdx+1)%2, PromptNone, nil)
					action, ok := readFrom(activeIdx)
					if !ok {
						handleDisconnect(activeIdx)
						return
					}
					if action.Type == ActionSelectAttackers && len(action.Attackers) > 0 {
						performAttack(g, action.Attackers, addLog)
					} else {
						addLog(fmt.Sprintf("%s chooses not to attack", g.Players[activeIdx].Name()))
					}
				}
				if len(g.Combat.Groups) > 0 {
					if !runPriorityLoop(false) {
						return
					}
				}

			case core.DeclareBlockers:
				if len(g.Combat.Groups) == 0 {
					continue
				}
				nonActiveIdx := (g.ActivePlayer + 1) % 2
				eligible := getEligibleBlockers(g, g.Players[nonActiveIdx].PlayerID())
				if len(eligible) > 0 {
					sendTo(nonActiveIdx, PromptDeclareBlockers, blockerOptions(g, eligible))
					sendTo(g.ActivePlayer, PromptNone, nil)
					action, ok := readFrom(nonActiveIdx)
					if !ok {
						handleDisconnect(nonActiveIdx)
						return
					}
					if action.Type == ActionSelectBlockers && len(action.Blockers) > 0 {
						performBlock(g, action.Blockers, addLog)
					}
				}
				if len(g.Combat.Groups) > 0 {
					if !runPriorityLoop(false) {
						return
					}
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
					for _, p := range g.Players {
						addLog(fmt.Sprintf("%s: %d life", p.Name(), p.Life()))
					}
					broadcast()
				}

			case core.EndCombat:
				g.Combat.Reset()

			case core.PostcombatMain:
				if !runPriorityLoop(true) {
					return
				}

			case core.EndStep:
				g.FireEvent(core.GameEvent{
					Type:     core.EvtEndStep,
					PlayerID: g.ActivePlayerObj().PlayerID(),
				})
				g.PutTriggersOnStack()
				if !g.Stack.IsEmpty() {
					if !runPriorityLoop(false) {
						return
					}
				}

			case core.Cleanup:
				g.DoCleanup()
			}

			g.CheckStateBasedActions()
			if g.IsGameOver() {
				broadcast()
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
