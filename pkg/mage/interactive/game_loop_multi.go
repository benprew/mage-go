package interactive

import (
	"fmt"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
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

	// disconnected tracks whether a player has disconnected so we can abort cleanly.
	disconnected := false

	// handleDisconnect sends a game-over message to the surviving player.
	handleDisconnect := func(disconnectedIdx int) {
		disconnected = true
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

	// --- Install priority handler on the engine ---
	g.OnPriority = func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		if disconnected {
			return mage.PriorityAction{Type: mage.PriorityPass}
		}
		action, ok := getAction(playerIdx, mainPhase)
		if !ok {
			handleDisconnect(playerIdx)
			return mage.PriorityAction{Type: mage.PriorityPass}
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
		broadcast()
	}

	// Log stack resolution
	g.BeforeStackResolve = func(g *mage.Game) {
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
			if disconnected {
				return
			}

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
				broadcast()
			case core.Draw:
				addLog(fmt.Sprintf("%s draws a card", g.ActivePlayerObj().Name()))
			case core.CombatDamage:
				if len(g.Combat.Groups) > 0 {
					for _, p := range g.Players {
						addLog(fmt.Sprintf("%s: %d life", p.Name(), p.Life()))
					}
					broadcast()
				}
			}

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
