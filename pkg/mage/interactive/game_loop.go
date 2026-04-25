package interactive

import (
	"fmt"
	"time"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// RunGameLoop is the main interactive game loop running in a goroutine.
// RunGameLoop runs the game until it ends, communicating state to/from the TUI
// via channels embedded in the HumanPlayer at g.Players[humanIdx].
// aiActionPause is how long to pause after each visible AI action so the player
// can see what happened; 0 means no pause (useful for testing).
func RunGameLoop(g *mage.Game, humanIdx int, aiActionPause time.Duration) {
	hp := g.PlayerAt(humanIdx).(*HumanPlayer)
	defer close(hp.toTUI)
	defer close(hp.choiceReqs)

	aiIdx := (humanIdx + 1) % 2
	aiPlayer, isAI := g.PlayerAt(aiIdx).(AutoPlayer)
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
		playerID := g.PlayerAt(humanIdx).PlayerID()
		options := GetAvailableActions(g, playerID)
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
		return aiPlayer.GetPriorityAction(g, g.GetLandsPlayedThisTurn(), mainPhase)
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
	g.SetOnPriority(func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		action := getAction(playerIdx, mainPhase)

		// Handle undo (loop until real action)
		for action.Type == ActionUndo && lastUndo.valid {
			restoreFromUndo(g, g.PlayerAt(humanIdx).PlayerID(), lastUndo)
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
				lastUndo = captureForUndo(g, g.PlayerAt(humanIdx).PlayerID(), len(gameLog))
			} else {
				lastUndo.valid = false
			}
		}

		return convertToEngineAction(action)
	})

	// Log actions after they execute
	g.SetAfterPriorityAction(func(g *mage.Game, playerIdx int, action mage.PriorityAction) {
		p := g.PlayerAt(playerIdx)
		switch action.Type {
		case mage.PriorityPlayLand:
			perm := g.FindPermanent(action.CardID)
			name := "a land"
			if perm != nil {
				name = perm.Name()
			}
			addLog(fmt.Sprintf("%s plays %s", p.Name(), name))
		case mage.PriorityCastSpell:
			obj := g.StackPeek()
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
	})

	// Log damage dealt by creatures and spells
	g.SetOnDamageDealt(func(sourceName, targetName string, amount int, isCombat bool) {
		addLog(fmt.Sprintf("  %s deals %d damage to %s", sourceName, amount, targetName))
	})

	// Log stack resolution
	g.SetBeforeStackResolve(func(g *mage.Game) {
		lastUndo.valid = false
		top := g.StackPeek()
		if top == nil {
			return
		}
		topName := "ability"
		if top.Card != nil {
			topName = top.Card.Name()
		}
		addLog(fmt.Sprintf("Resolving %s%s", topName, targetSuffix(g, top.Targets)))
	})

	// --- Main turn loop ---
	for g.CurrentTurn() <= 100 {
		for _, step := range core.AllSteps() {
			// Pre-step logging
			switch step {
			case core.CombatDamage:
				if len(g.CombatGroups()) > 0 {
					addLog("── Combat damage ──")
					logCombatPreview(g, addLog)
				}
			}

			g.RunStepWithPriority(step)

			// Post-step logging and display
			switch step {
			case core.Untap:
				addLog(fmt.Sprintf("── Turn %d: %s ──", g.CurrentTurn(), g.ActivePlayerObj().Name()))
				if g.ActivePlayerIndex() != humanIdx {
					send(PromptNone, nil)
				}
			case core.Draw:
				if g.ActivePlayerIndex() == humanIdx {
					addLog("You draw a card")
				} else {
					addLog(fmt.Sprintf("%s draws a card", g.ActivePlayerObj().Name()))
				}
			case core.CombatDamage:
				if len(g.CombatGroups()) > 0 {
					reportCombatResults(g, addLog)
					showAIAction()
				}
			}

			if g.IsGameOver() {
				send(PromptNone, nil)
				return
			}
		}

		if g.HasExtraTurns() {
			extraPlayerID, _ := g.PopExtraTurn()
			for i, p := range g.AllPlayers() {
				if p.PlayerID() == extraPlayerID {
					g.SetActivePlayerIndex(i)
					break
				}
			}
		} else {
			g.SetActivePlayerIndex((g.ActivePlayerIndex() + 1) % g.PlayerCount())
		}
		g.SetTurn(g.CurrentTurn() + 1)
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
