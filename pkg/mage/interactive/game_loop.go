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

	// --- Install priority handler on the engine ---
	g.OnPriority = func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		action := getAction(playerIdx, mainPhase)

		// Handle undo (loop until real action)
		for action.Type == ActionUndo && lastUndo.valid {
			restoreFromUndo(g, g.Players[humanIdx].PlayerID(), lastUndo)
			if len(gameLog) > lastUndo.logLen {
				gameLog = gameLog[:lastUndo.logLen]
			}
			lastUndo.valid = false
			addLog("Undid last action")
			action = getAction(playerIdx, mainPhase)
		}

		if action.Type != ActionPass {
			// Capture undo state before action executes
			if playerIdx == humanIdx {
				lastUndo = captureForUndo(g, g.Players[humanIdx].PlayerID(), len(gameLog))
			} else {
				lastUndo.valid = false
			}
		}

		return convertToEngineAction(action)
	}

	// Log actions after they execute
	g.AfterPriorityAction = func(g *mage.Game, playerIdx int, action mage.PriorityAction) {
		p := g.Players[playerIdx]
		switch action.Type {
		case mage.PriorityPlayLand:
			perm := g.FindPermanent(action.CardID)
			name := "a land"
			if perm != nil {
				name = perm.Name()
			}
			addLog(fmt.Sprintf("%s plays %s", p.Name(), name))
		case mage.PriorityCastSpell:
			obj := g.Stack.Peek()
			name := "a spell"
			if obj != nil && obj.Card != nil {
				name = obj.Card.Name()
			}
			addLog(fmt.Sprintf("%s casts %s%s", p.Name(), name, targetSuffix(g, action.Targets)))
		case mage.PriorityActivateAbility:
			perm := g.FindPermanent(action.PermanentID)
			name := "permanent"
			if perm != nil {
				name = perm.Name()
			}
			addLog(fmt.Sprintf("Activated ability: %s%s", name, targetSuffix(g, action.Targets)))
		}
		if playerIdx != humanIdx {
			showAIAction()
		}
	}

	// Log stack resolution
	g.BeforeStackResolve = func(g *mage.Game) {
		lastUndo.valid = false
		top := g.Stack.Peek()
		if top == nil {
			return
		}
		topName := "ability"
		if top.Card != nil {
			topName = top.Card.Name()
		}
		addLog(fmt.Sprintf("Resolving %s%s", topName, targetSuffix(g, top.Targets)))
	}

	// --- Main turn loop ---
	for g.Turn <= 100 {
		for _, step := range core.AllSteps() {
			// Pre-step logging
			switch step {
			case core.CombatDamage:
				if len(g.Combat.Groups) > 0 {
					addLog("── Combat damage ──")
					logCombatPreview(g, addLog)
				}
			}

			g.RunStepWithPriority(step)

			// Post-step logging and display
			switch step {
			case core.Untap:
				addLog(fmt.Sprintf("── Turn %d: %s ──", g.Turn, g.ActivePlayerObj().Name()))
				if g.ActivePlayer != humanIdx {
					send(PromptNone, nil)
				}
			case core.Draw:
				if g.ActivePlayer == humanIdx {
					addLog("You draw a card")
				} else {
					addLog(fmt.Sprintf("%s draws a card", g.ActivePlayerObj().Name()))
				}
			case core.CombatDamage:
				if len(g.Combat.Groups) > 0 {
					reportCombatResults(g, humanIdx, addLog)
					showAIAction()
				}
			}

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

// convertToEngineAction converts an interactive PriorityAction to an engine PriorityAction.
func convertToEngineAction(action PriorityAction) mage.PriorityAction {
	switch action.Type {
	case ActionPlayLand:
		return mage.PriorityAction{
			Type:   mage.PriorityPlayLand,
			CardID: action.CardID,
		}
	case ActionCastSpell:
		return mage.PriorityAction{
			Type:    mage.PriorityCastSpell,
			CardID:  action.CardID,
			Targets: action.Targets,
			XValue:  action.XValue,
		}
	case ActionActivateAbility:
		return mage.PriorityAction{
			Type:        mage.PriorityActivateAbility,
			PermanentID: action.PermanentID,
			AbilityIdx:  action.AbilityIndex,
			Targets:     action.Targets,
		}
	default:
		return mage.PriorityAction{Type: mage.PriorityPass}
	}
}
