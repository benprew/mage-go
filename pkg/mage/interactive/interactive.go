package interactive

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// PlayerChannels bundles the communication channels between the game loop and one player's TUI.
type PlayerChannels struct {
	ToPlayer    chan<- GameMsg
	FromPlayer  <-chan PriorityAction
	ChoiceReqs  chan<- ChoiceRequest
	ChoiceResps <-chan ChoiceResponse
}

//go:generate enumer -type=ActionType -trimprefix=Action -output=interactive_enumer.go

// ActionType identifies what kind of action a player is taking.
type ActionType int

const (
	ActionPass ActionType = iota
	ActionPlayLand
	ActionCastSpell
	ActionActivateAbility
	ActionSelectAttackers
	ActionSelectBlockers
	ActionUndo
)

// PriorityAction is a decision sent from the TUI (or AI) to the game loop.
type PriorityAction struct {
	Type         ActionType
	CardID       uuid.UUID
	CardName     string
	Targets      []uuid.UUID
	PermanentID  uuid.UUID
	AbilityIndex int
	XValue       int
	Attackers    []uuid.UUID
	Blockers     []mage.BlockAssignment
}

// PromptType tells the TUI what kind of input is needed.
type PromptType int

const (
	PromptNone PromptType = iota
	PromptMainPhaseAction
	PromptPriority
	PromptDeclareAttackers
	PromptDeclareBlockers
	PromptChooseTargets
)

func (pt PromptType) String() string {
	switch pt {
	case PromptMainPhaseAction:
		return "Main Phase"
	case PromptPriority:
		return "Priority"
	case PromptDeclareAttackers:
		return "Declare Attackers"
	case PromptDeclareBlockers:
		return "Declare Blockers"
	case PromptChooseTargets:
		return "Choose Targets"
	default:
		return ""
	}
}

// ActionOption describes one available action in the TUI menu.
type ActionOption struct {
	Type         ActionType
	Label        string
	CardName     string // raw card/permanent name, without cost prefix
	CardID       uuid.UUID
	PermanentID  uuid.UUID
	AbilityIndex int
	NeedsTarget  bool
	TargetType   mage.Target
	ManaCost     string
}

// GameMsg is sent from the game goroutine to the TUI.
type GameMsg struct {
	State    *GameState
	Prompt   PromptType
	Options  []ActionOption
	Log      []string
	GameOver bool
	Winner   string
	CanUndo  bool
}

type undoSnapshot struct {
	valid          bool
	hand           []mage.Card
	manaPool       []core.Mana
	landsPlayed    int
	battlefieldLen int
	stackSize      int
	tappedState    map[uuid.UUID]bool
	logLen         int
}

func captureForUndo(g *mage.Game, playerID uuid.UUID, logLen int) undoSnapshot {
	p := g.GetPlayer(playerID)
	if p == nil {
		return undoSnapshot{}
	}
	hand := make([]mage.Card, len(p.Hand()))
	copy(hand, p.Hand())
	pool := p.ManaPool().SnapshotPool()
	tapped := make(map[uuid.UUID]bool)
	for _, perm := range g.Battlefield {
		tapped[perm.ID()] = perm.Tapped
	}
	return undoSnapshot{
		valid:          true,
		hand:           hand,
		manaPool:       pool,
		landsPlayed:    g.LandsPlayedThisTurn,
		battlefieldLen: len(g.Battlefield),
		stackSize:      g.Stack.Size(),
		tappedState:    tapped,
		logLen:         logLen,
	}
}

func restoreFromUndo(g *mage.Game, playerID uuid.UUID, snap undoSnapshot) {
	p := g.GetPlayer(playerID)
	if p == nil || !snap.valid {
		return
	}
	p.SetHand(snap.hand)
	p.ManaPool().RestorePool(snap.manaPool)
	g.LandsPlayedThisTurn = snap.landsPlayed
	if len(g.Battlefield) > snap.battlefieldLen {
		g.Battlefield = g.Battlefield[:snap.battlefieldLen]
	}
	for g.Stack.Size() > snap.stackSize {
		g.Stack.Pop()
	}
	for _, perm := range g.Battlefield {
		if was, ok := snap.tappedState[perm.ID()]; ok {
			perm.Tapped = was
		}
	}
}

// GameState is a read-only snapshot of the game for the TUI.
type GameState struct {
	Turn         int
	Step         string
	ActivePlayer string
	You          PlayerState
	Opponent     PlayerState
	StackItems   []StackItemState
}

// PlayerState is a snapshot of a player.
type PlayerState struct {
	ID             uuid.UUID
	Name           string
	Life           int
	HandCount      int
	Hand           []CardState
	Battlefield    []PermanentState
	Graveyard      []CardState
	GraveyardCount int
	ManaPool       ManaPoolState
	LibraryCount   int
}

// PermanentState is a snapshot of a permanent.
type PermanentState struct {
	ID         uuid.UUID
	Name       string
	Power      int
	Toughness  int
	Tapped     bool
	SummonSick bool
	IsCreature bool
	IsLand     bool
	IsArtifact bool
	Attacking  bool
	Counters   map[string]int
	Keywords   []string
	ManaCost   string
	Types      string
	SubTypes   string
	RulesText  string
}

// CardState is a snapshot of a card in hand.
type CardState struct {
	ID        uuid.UUID
	Name      string
	ManaCost  string
	IsLand    bool
	Types     string
	SubTypes  string
	Power     int
	Toughness int
	RulesText string
}

// ManaPoolState is a snapshot of a mana pool.
type ManaPoolState struct {
	White     int
	Blue      int
	Black     int
	Red       int
	Green     int
	Colorless int
}

// StackItemState is a snapshot of a stack object.
type StackItemState struct {
	Name       string
	Controller string
	IsAbility  bool
	Targets    []string // human-readable target names
}

// ChoiceType identifies what kind of interactive choice the human player must make.
type ChoiceType int

const (
	ChoicePermanent      ChoiceType = iota // pick one permanent from candidates
	ChoiceCardsFromHand                    // pick N cards from hand (multi-select)
	ChoiceManaColor                        // pick a mana color
	ChoiceCardFromLibrary                  // pick one card from library candidates
	ChoiceMay                              // yes/no for an optional ability
	ChoiceMode                             // pick one mode from a modal spell/ability
)

// ChoiceOption is one selectable item in a ChoiceRequest.
type ChoiceOption struct {
	ID    uuid.UUID  // permanent or card ID (zero for color/boolean options)
	Label string
	Color core.Color // populated for ChoiceManaColor options
}

// ChoiceRequest is sent by HumanPlayer to the TUI when a player decision is needed
// during game resolution.
type ChoiceRequest struct {
	Type    ChoiceType
	Reason  string
	Amount  int          // for ChoiceCardsFromHand: how many to select
	Options []ChoiceOption
}

// ChoiceResponse is sent by the TUI back to the waiting HumanPlayer.
type ChoiceResponse struct {
	SelectedIDs   []uuid.UUID // for permanent/card choices
	SelectedColor core.Color  // for ChoiceManaColor
	Accepted      bool        // for ChoiceMay: true = yes
	SelectedIndex int         // for ChoiceMode: index of chosen mode
}

func buildRulesText(c mage.Card) string {
	var parts []string
	// Keywords come from the card's attr seeds (canonical storage).
	for a, count := range c.AttrSeeds() {
		if count > 0 && core.IsKeywordAttr(a) {
			parts = append(parts, a.String())
		}
	}
	for _, a := range c.Abilities() {
		switch ab := a.(type) {
		case *mage.ProtectionAbility:
			var colors []string
			for _, col := range ab.FromColors {
				colors = append(colors, col.String())
			}
			parts = append(parts, fmt.Sprintf("Protection from %s", joinStrings(colors)))
		case *mage.SpellAbility:
			for _, eff := range ab.Effects() {
				parts = append(parts, eff.Text())
			}
		case mage.ActivatedAbility:
			var costParts []string
			for _, cost := range ab.Costs() {
				costParts = append(costParts, cost.Text())
			}
			var effParts []string
			for _, eff := range ab.Effects() {
				effParts = append(effParts, eff.Text())
			}
			parts = append(parts, fmt.Sprintf("%s: %s", joinStrings(costParts), joinStrings(effParts)))
		}
	}
	return joinStrings(parts)
}

func joinStrings(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

// SnapshotGameState creates a read-only snapshot of the game for the TUI.
func SnapshotGameState(g *mage.Game, humanIndex int) *GameState {
	human := g.Players[humanIndex]
	aiIndex := (humanIndex + 1) % 2
	ai := g.Players[aiIndex]

	return &GameState{
		Turn:         g.Turn,
		Step:         g.Step.String(),
		ActivePlayer: g.ActivePlayerObj().Name(),
		You:          snapshotPlayer(g, human, true),
		Opponent:     snapshotPlayer(g, ai, false),
		StackItems:   snapshotStack(g),
	}
}

func snapshotPlayer(g *mage.Game, p mage.Player, showHand bool) PlayerState {
	ps := PlayerState{
		ID:             p.PlayerID(),
		Name:           p.Name(),
		Life:           p.Life(),
		HandCount:      len(p.Hand()),
		GraveyardCount: len(p.Graveyard()),
		LibraryCount:   len(p.Library()),
		ManaPool:       snapshotManaPool(p.ManaPool()),
	}

	if showHand {
		for _, c := range p.Hand() {
			cs := CardState{
				ID:        c.ID(),
				Name:      c.Name(),
				ManaCost:  c.ManaCost().String(),
				IsLand:    c.HasType(core.TypeLand),
				Power:     c.Power(),
				Toughness: c.Toughness(),
				RulesText: buildRulesText(c),
			}
			for _, t := range c.Types() {
				if cs.Types != "" {
					cs.Types += " "
				}
				cs.Types += t.String()
			}
			for _, st := range c.SubTypes() {
				if cs.SubTypes != "" {
					cs.SubTypes += " "
				}
				cs.SubTypes += st
			}
			ps.Hand = append(ps.Hand, cs)
		}
	}

	for _, c := range p.Graveyard() {
		cs := CardState{
			ID:        c.ID(),
			Name:      c.Name(),
			ManaCost:  c.ManaCost().String(),
			IsLand:    c.HasType(core.TypeLand),
			Power:     c.Power(),
			Toughness: c.Toughness(),
			RulesText: buildRulesText(c),
		}
		for _, t := range c.Types() {
			if cs.Types != "" {
				cs.Types += " "
			}
			cs.Types += t.String()
		}
		for _, st := range c.SubTypes() {
			if cs.SubTypes != "" {
				cs.SubTypes += " "
			}
			cs.SubTypes += st
		}
		ps.Graveyard = append(ps.Graveyard, cs)
	}

	for _, perm := range g.Battlefield {
		if perm.Controller != p.PlayerID() {
			continue
		}
		permState := PermanentState{
			ID:         perm.ID(),
			Name:       perm.Name(),
			Power:      perm.CurrentPower(g),
			Toughness:  perm.CurrentToughness(g),
			Tapped:     perm.Tapped,
			SummonSick: perm.HasAttr(core.AttrSummonSick),
			IsCreature: perm.HasType(core.TypeCreature),
			IsLand:     perm.HasType(core.TypeLand),
			IsArtifact: perm.HasType(core.TypeArtifact),
			Attacking:  g.Combat.IsAttacking(perm.ID()),
			ManaCost:   perm.Card.ManaCost().String(),
			RulesText:  buildRulesText(perm.Card),
		}
		for _, t := range perm.Card.Types() {
			if permState.Types != "" {
				permState.Types += " "
			}
			permState.Types += t.String()
		}
		for _, st := range perm.Card.SubTypes() {
			if permState.SubTypes != "" {
				permState.SubTypes += " "
			}
			permState.SubTypes += st
		}
		if len(perm.Counters) > 0 {
			permState.Counters = make(map[string]int)
			for ct, n := range perm.Counters {
				permState.Counters[ct.String()] = n
			}
		}
		permState.Keywords = perm.KeywordNames()
		ps.Battlefield = append(ps.Battlefield, permState)
	}

	return ps
}

func snapshotManaPool(mp *mage.ManaPool) ManaPoolState {
	return ManaPoolState{
		White:     mp.Count(core.White),
		Blue:      mp.Count(core.Blue),
		Black:     mp.Count(core.Black),
		Red:       mp.Count(core.Red),
		Green:     mp.Count(core.Green),
		Colorless: mp.Count(core.Colorless),
	}
}

func snapshotStack(g *mage.Game) []StackItemState {
	var items []StackItemState
	for _, obj := range g.Stack.Objects() {
		name := "Ability"
		if obj.Card != nil {
			name = obj.Card.Name()
		}
		controller := ""
		p := g.GetPlayer(obj.Controller)
		if p != nil {
			controller = p.Name()
		}
		var targetNames []string
		for _, tid := range obj.Targets {
			targetNames = append(targetNames, resolveTargetName(g, tid))
		}
		items = append(items, StackItemState{
			Name:       name,
			Controller: controller,
			IsAbility:  obj.IsAbility,
			Targets:    targetNames,
		})
	}
	return items
}

// GetAvailableActions returns the actions available to a player right now.
func GetAvailableActions(g *mage.Game, playerID uuid.UUID, landsPlayed int, mainPhase bool) []ActionOption {
	var options []ActionOption

	if mainPhase {
		if landsPlayed < 1 {
			p := g.GetPlayer(playerID)
			if p != nil {
				for _, c := range p.Hand() {
					if c.HasType(core.TypeLand) {
						options = append(options, ActionOption{
							Type:   ActionPlayLand,
							Label:  fmt.Sprintf("Play %s", c.Name()),
							CardID: c.ID(),
							CardName: c.Name(),
						})
					}
				}
			}
		}

		for _, card := range g.GetCastableSpells(playerID) {
			needsTarget := false
			var targetType mage.Target
			for _, a := range card.Abilities() {
				if sa, ok := a.(*mage.SpellAbility); ok {
					for _, t := range sa.Targets() {
						needsTarget = true
						targetType = t
						break
					}
				}
			}
			options = append(options, ActionOption{
				Type:        ActionCastSpell,
				Label:       fmt.Sprintf("Cast %s %s", card.Name(), card.ManaCost()),
				CardID:      card.ID(),
				CardName:    card.Name(),
				NeedsTarget: needsTarget,
				TargetType:  targetType,
				ManaCost:    card.ManaCost().String(),
			})
		}
	} else {
		p := g.GetPlayer(playerID)
		if p != nil {
			for _, card := range p.Hand() {
				if !card.HasType(core.TypeInstant) {
					continue
				}
				if !g.CanAfford(playerID, card.ManaCost()) {
					continue
				}
				needsTarget := false
				var targetType mage.Target
				for _, a := range card.Abilities() {
					if sa, ok := a.(*mage.SpellAbility); ok {
						for _, t := range sa.Targets() {
							needsTarget = true
							targetType = t
							break
						}
					}
				}
				options = append(options, ActionOption{
					Type:        ActionCastSpell,
					Label:       fmt.Sprintf("Cast %s %s", card.Name(), card.ManaCost()),
					CardID:      card.ID(),
					NeedsTarget: needsTarget,
					CardName:    card.Name(),
					TargetType:  targetType,
					ManaCost:    card.ManaCost().String(),
				})
			}
		}
	}

	for _, info := range g.GetActivatableAbilities(playerID) {
		options = append(options, ActionOption{
			Type:         ActionActivateAbility,
			Label:        fmt.Sprintf("Activate %s: %s", info.PermanentName, info.Description),
			PermanentID:  info.PermanentID,
			AbilityIndex: info.AbilityIndex,
		})
	}

	options = append(options, ActionOption{
		Type:  ActionPass,
		Label: "Pass",
	})

	return options
}

// GetTargetChoices builds the list of valid target IDs and display labels for
// a spell or ability that needs a target. It always includes both players (with
// real UUIDs from the snapshot) and any creatures on the battlefield.
func GetTargetChoices(state *GameState, opt ActionOption) ([]uuid.UUID, []string) {
	var ids []uuid.UUID
	var labels []string

	// Creatures from both sides
	for _, p := range state.Opponent.Battlefield {
		if p.IsCreature {
			ids = append(ids, p.ID)
			labels = append(labels, fmt.Sprintf("%s %d/%d (%s)", p.Name, p.Power, p.Toughness, state.Opponent.Name))
		}
	}
	for _, p := range state.You.Battlefield {
		if p.IsCreature {
			ids = append(ids, p.ID)
			labels = append(labels, fmt.Sprintf("%s %d/%d (You)", p.Name, p.Power, p.Toughness))
		}
	}

	// Both players — only if we have real UUIDs
	if state.Opponent.ID != uuid.Nil {
		ids = append(ids, state.Opponent.ID)
		labels = append(labels, fmt.Sprintf("%s (player)", state.Opponent.Name))
	}
	if state.You.ID != uuid.Nil {
		ids = append(ids, state.You.ID)
		labels = append(labels, "You (player)")
	}

	return ids, labels
}

// findPlayerIndex returns the index of the player with the given ID in g.Players.
func findPlayerIndex(g *mage.Game, id uuid.UUID) int {
	for i, p := range g.Players {
		if p.PlayerID() == id {
			return i
		}
	}
	return 0
}

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

func describeAction(action PriorityAction) string {
	switch action.Type {
	case ActionPlayLand:
		return fmt.Sprintf("plays %s", action.CardName)
	case ActionCastSpell:
		return fmt.Sprintf("casts %s", action.CardName)
	case ActionActivateAbility:
		return fmt.Sprintf("activates ability on %s", action.CardName)
	default:
		return "passes"
	}
}

// resolveTargetName returns a human-readable name for a target UUID (permanent or player).
func resolveTargetName(g *mage.Game, id uuid.UUID) string {
	if perm := g.FindPermanent(id); perm != nil {
		return perm.Name()
	}
	if player := g.GetPlayer(id); player != nil {
		return player.Name()
	}
	return "unknown"
}

// targetSuffix builds " targeting X, Y" for a list of target IDs, or "" if none.
func targetSuffix(g *mage.Game, targets []uuid.UUID) string {
	if len(targets) == 0 {
		return ""
	}
	names := make([]string, len(targets))
	for i, id := range targets {
		names[i] = resolveTargetName(g, id)
	}
	return " targeting " + strings.Join(names, ", ")
}

func executeAction(g *mage.Game, playerID uuid.UUID, action PriorityAction, addLog func(string)) {
	var err error
	switch action.Type {
	case ActionPlayLand:
		err = g.PlayLand(playerID, action.CardID)
		if err == nil {
			p := g.GetPlayer(playerID)
			name := action.CardName
			if name == "" && p != nil {
				name = "a land"
			}
			addLog(fmt.Sprintf("%s plays %s", p.Name(), name))
		}
	case ActionCastSpell:
		err = g.CastSpellByID(playerID, action.CardID, action.Targets, action.XValue)
		if err == nil {
			p := g.GetPlayer(playerID)
			// Use the card name from the stack object — action.CardName may be a
			// display label like "Cast Giant Growth {G}" from the TUI option.
			name := action.CardName
			obj := g.Stack.Peek()
			if obj != nil && obj.Card != nil {
				name = obj.Card.Name()
			}
			addLog(fmt.Sprintf("%s casts %s%s", p.Name(), name, targetSuffix(g, action.Targets)))
		}
	case ActionActivateAbility:
		err = g.ActivateAbilityByIndex(playerID, action.PermanentID, action.AbilityIndex, action.Targets)
		if err == nil {
			addLog(fmt.Sprintf("Activated ability: %s%s", action.CardName, targetSuffix(g, action.Targets)))
		}
	}
	if err != nil {
		addLog(fmt.Sprintf("Error: %v", err))
	}
}

func getEligibleAttackers(g *mage.Game, playerID uuid.UUID) []*mage.Permanent {
	var eligible []*mage.Permanent
	for _, perm := range g.Battlefield {
		if perm.Controller != playerID || !perm.CanDeclareAsAttacker(g) {
			continue
		}
		eligible = append(eligible, perm)
	}
	return eligible
}

func attackerOptions(eligible []*mage.Permanent) []ActionOption {
	var options []ActionOption
	for _, perm := range eligible {
		options = append(options, ActionOption{
			Type:        ActionSelectAttackers,
			Label:       fmt.Sprintf("%s %d/%d", perm.Name(), perm.Card.Power(), perm.Card.Toughness()),
			PermanentID: perm.ID(),
		})
	}
	return options
}

func performAttack(g *mage.Game, attackerIDs []uuid.UUID, addLog func(string)) {
	defender := g.NonActivePlayerObj()
	active := g.ActivePlayerObj()
	for _, id := range attackerIDs {
		atk := g.FindPermanent(id)
		if atk == nil {
			continue
		}
		if !atk.HasKeyword(core.Vigilance) {
			atk.Tapped = true
		}
		g.Combat.AddAttacker(id, defender.PlayerID())
		addLog(fmt.Sprintf("%s attacks with %s %d/%d",
			active.Name(), atk.Name(), atk.CurrentPower(g), atk.CurrentToughness(g)))
		g.FireEvent(core.GameEvent{
			Type:     core.EvtDeclaredAttacker,
			SourceID: id,
			PlayerID: active.PlayerID(),
		})
	}
}

func getEligibleBlockers(g *mage.Game, playerID uuid.UUID) []*mage.Permanent {
	var eligible []*mage.Permanent
	for _, perm := range g.Battlefield {
		if perm.Controller != playerID || !perm.CanDeclareAsBlocker(g) {
			continue
		}
		eligible = append(eligible, perm)
	}
	return eligible
}

func blockerOptions(g *mage.Game, eligible []*mage.Permanent) []ActionOption {
	var options []ActionOption
	for _, perm := range eligible {
		options = append(options, ActionOption{
			Type:        ActionSelectBlockers,
			Label:       fmt.Sprintf("%s %d/%d", perm.Name(), perm.CurrentPower(g), perm.CurrentToughness(g)),
			PermanentID: perm.ID(),
		})
	}
	options = append(options, ActionOption{
		Type:  ActionPass,
		Label: "Done (confirm blocks)",
	})
	return options
}

func performBlock(g *mage.Game, blockers []mage.BlockAssignment, addLog func(string)) {
	nonActive := g.NonActivePlayerObj()
	for _, ba := range blockers {
		blocker := g.FindPermanent(ba.BlockerID)
		attacker := g.FindPermanent(ba.AttackerID)
		if blocker == nil || attacker == nil {
			continue
		}
		if !mage.CanBlock(blocker, attacker, g) {
			continue
		}
		if mage.HasLandwalkEvasion(attacker, nonActive.PlayerID(), g) {
			continue
		}
		g.Combat.AddBlocker(ba.BlockerID, ba.AttackerID)
		addLog(fmt.Sprintf("%s blocks %s with %s",
			nonActive.Name(), attacker.Name(), blocker.Name()))
		g.FireEvent(core.GameEvent{
			Type:     core.EvtDeclaredBlocker,
			SourceID: ba.BlockerID,
			TargetID: ba.AttackerID,
			PlayerID: nonActive.PlayerID(),
		})
	}
}

func reportCombatResults(g *mage.Game, humanIdx int, addLog func(string)) {
	for _, p := range g.Players {
		addLog(fmt.Sprintf("%s: %d life", p.Name(), p.Life()))
	}
}

// logCombatPreview logs a summary of each combat group before damage is dealt.
func logCombatPreview(g *mage.Game, addLog func(string)) {
	for _, grp := range g.Combat.Groups {
		atk := g.FindPermanent(grp.AttackerID)
		if atk == nil {
			continue
		}
		defender := g.GetPlayer(grp.DefenderID)
		defName := "player"
		if defender != nil {
			defName = defender.Name()
		}
		if len(grp.BlockerIDs) == 0 {
			addLog(fmt.Sprintf("  %s (%d/%d) → %s unblocked",
				atk.Name(), atk.Card.Power(), atk.Card.Toughness(), defName))
		} else {
			parts := make([]string, 0, len(grp.BlockerIDs))
			for _, bid := range grp.BlockerIDs {
				blk := g.FindPermanent(bid)
				if blk != nil {
					parts = append(parts, fmt.Sprintf("%s (%d/%d)", blk.Name(), blk.Card.Power(), blk.Card.Toughness()))
				}
			}
			addLog(fmt.Sprintf("  %s (%d/%d) blocked by %s",
				atk.Name(), atk.Card.Power(), atk.Card.Toughness(), strings.Join(parts, ", ")))
		}
	}
}

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
