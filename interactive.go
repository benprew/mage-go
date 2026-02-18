package mage

import (
	"fmt"

	"github.com/google/uuid"
)

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
	Type          ActionType
	CardID        uuid.UUID
	CardName      string
	Targets       []uuid.UUID
	PermanentID   uuid.UUID
	AbilityIndex  int
	XValue        int
	Attackers     []uuid.UUID
	Blockers      map[uuid.UUID]uuid.UUID // blocker -> attacker
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
	Type          ActionType
	Label         string
	CardID        uuid.UUID
	PermanentID   uuid.UUID
	AbilityIndex  int
	NeedsTarget   bool
	TargetType    Target // for target selection
	ManaCost      string
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

// undoSnapshot captures game state before a human action for potential undo.
type undoSnapshot struct {
	valid          bool
	hand           []Card
	manaPool       []Mana
	landsPlayed    int
	battlefieldLen int
	stackSize      int
	tappedState    map[uuid.UUID]bool
	logLen         int
}

func captureForUndo(g *Game, playerID uuid.UUID, logLen int) undoSnapshot {
	p := g.GetPlayer(playerID)
	if p == nil {
		return undoSnapshot{}
	}
	hand := make([]Card, len(p.Hand()))
	copy(hand, p.Hand())
	pool := make([]Mana, len(p.ManaPool().pool))
	copy(pool, p.ManaPool().pool)
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

func restoreFromUndo(g *Game, playerID uuid.UUID, snap undoSnapshot) {
	p := g.GetPlayer(playerID)
	if p == nil || !snap.valid {
		return
	}
	p.SetHand(snap.hand)
	p.ManaPool().pool = snap.manaPool
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
	Name           string
	Life           int
	HandCount      int
	Hand           []CardState
	Battlefield    []PermanentState
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
	RulesText string // ability descriptions
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
}

// buildRulesText generates a human-readable rules text from a card's abilities.
func buildRulesText(c Card) string {
	var parts []string
	for _, a := range c.Abilities() {
		switch ab := a.(type) {
		case *KeywordAbility:
			parts = append(parts, ab.Keyword.String())
		case *ProtectionAbility:
			var colors []string
			for _, col := range ab.FromColors {
				colors = append(colors, col.String())
			}
			parts = append(parts, fmt.Sprintf("Protection from %s", joinStrings(colors)))
		case *SpellAbility:
			for _, eff := range ab.Effects() {
				parts = append(parts, eff.Text())
			}
		case ActivatedAbility:
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
func SnapshotGameState(g *Game, humanIndex int) *GameState {
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

func snapshotPlayer(g *Game, p Player, showHand bool) PlayerState {
	ps := PlayerState{
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
				IsLand:    c.HasType(TypeLand),
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
			SummonSick: perm.SummonSick,
			IsCreature: perm.HasType(TypeCreature),
			IsLand:     perm.HasType(TypeLand),
			IsArtifact: perm.HasType(TypeArtifact),
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
		for _, a := range perm.RuntimeAbilities {
			if ka, ok := a.(*KeywordAbility); ok {
				permState.Keywords = append(permState.Keywords, ka.Keyword.String())
			}
		}
		ps.Battlefield = append(ps.Battlefield, permState)
	}

	return ps
}

func snapshotManaPool(mp *ManaPool) ManaPoolState {
	return ManaPoolState{
		White:     mp.Count(White),
		Blue:      mp.Count(Blue),
		Black:     mp.Count(Black),
		Red:       mp.Count(Red),
		Green:     mp.Count(Green),
		Colorless: mp.Count(Colorless),
	}
}

func snapshotStack(g *Game) []StackItemState {
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
		items = append(items, StackItemState{
			Name:       name,
			Controller: controller,
			IsAbility:  obj.IsAbility,
		})
	}
	return items
}

// GetAvailableActions returns the actions available to a player right now.
func GetAvailableActions(g *Game, playerID uuid.UUID, landsPlayed int, mainPhase bool) []ActionOption {
	var options []ActionOption

	if mainPhase {
		// Land drop
		if landsPlayed < 1 {
			p := g.GetPlayer(playerID)
			if p != nil {
				for _, c := range p.Hand() {
					if c.HasType(TypeLand) {
						options = append(options, ActionOption{
							Type:   ActionPlayLand,
							Label:  fmt.Sprintf("Play %s", c.Name()),
							CardID: c.ID(),
						})
					}
				}
			}
		}

		// Castable spells (at sorcery speed during main phase)
		for _, card := range g.GetCastableSpells(playerID) {
			needsTarget := false
			var targetType Target
			for _, a := range card.Abilities() {
				if sa, ok := a.(*SpellAbility); ok {
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
				TargetType:  targetType,
				ManaCost:    card.ManaCost().String(),
			})
		}
	} else {
		// Priority only: instants
		p := g.GetPlayer(playerID)
		if p != nil {
			for _, card := range p.Hand() {
				if !card.HasType(TypeInstant) {
					continue
				}
				if !g.CanAfford(playerID, card.ManaCost()) {
					continue
				}
				needsTarget := false
				var targetType Target
				for _, a := range card.Abilities() {
					if sa, ok := a.(*SpellAbility); ok {
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
					TargetType:  targetType,
					ManaCost:    card.ManaCost().String(),
				})
			}
		}
	}

	// Activated abilities (available in both main and priority)
	for _, info := range g.GetActivatableAbilities(playerID) {
		options = append(options, ActionOption{
			Type:         ActionActivateAbility,
			Label:        fmt.Sprintf("Activate %s: %s", info.PermanentName, info.Description),
			PermanentID:  info.PermanentID,
			AbilityIndex: info.AbilityIndex,
		})
	}

	// Pass is always an option
	options = append(options, ActionOption{
		Type:  ActionPass,
		Label: "Pass",
	})

	return options
}

// RunGameLoop is the main interactive game loop running in a goroutine.
func RunGameLoop(g *Game, humanIdx int, toTUI chan<- GameMsg, fromTUI <-chan PriorityAction) {
	defer close(toTUI)

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
	}

	send := func(prompt PromptType, options []ActionOption) {
		state := SnapshotGameState(g, humanIdx)
		toTUI <- GameMsg{
			State:    state,
			Prompt:   prompt,
			Options:  options,
			Log:      append([]string{}, gameLog...), // copy
			GameOver: g.IsGameOver(),
			Winner:   g.Winner(),
			CanUndo:  lastUndo.valid,
		}
	}

	// Get action from a player. AI computes inline; human gets a TUI prompt.
	getHumanAction := func(mainPhase bool) PriorityAction {
		playerID := g.Players[humanIdx].PlayerID()
		options := GetAvailableActions(g, playerID, g.LandsPlayedThisTurn, mainPhase)
		// Auto-pass if the only option is Pass AND no undo available
		if len(options) == 1 && options[0].Type == ActionPass && !lastUndo.valid {
			return PriorityAction{Type: ActionPass}
		}
		if mainPhase {
			send(PromptMainPhaseAction, options)
		} else {
			send(PromptPriority, options)
		}
		return <-fromTUI
	}

	getAIAction := func(mainPhase bool) PriorityAction {
		if !isAI {
			return PriorityAction{Type: ActionPass}
		}
		action := aiPlayer.GetPriorityAction(g, g.LandsPlayedThisTurn, mainPhase)
		if action.Type != ActionPass {
			addLog(fmt.Sprintf("AI: %s", describeAction(action)))
			// Send a state update so the TUI shows what the AI did
			send(PromptNone, nil)
		}
		return action
	}

	getAction := func(playerIdx int, mainPhase bool) PriorityAction {
		if playerIdx == humanIdx {
			return getHumanAction(mainPhase)
		}
		return getAIAction(mainPhase)
	}

	// Priority loop: both players get to act, stack resolves when both pass
	runPriorityLoop := func(mainPhase bool) {
		for {
			if g.IsGameOver() {
				return
			}

			activeIdx := g.ActivePlayer
			nonActiveIdx := (g.ActivePlayer + 1) % 2

			// Active player gets priority
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
					lastUndo.valid = false // AI acted, clear undo
				}
				executeAction(g, g.Players[activeIdx].PlayerID(), action, addLog)
				g.CheckStateBasedActions()
				if g.IsGameOver() {
					return
				}
				continue // restart priority — both must pass again
			}

			// Non-active player gets priority (always instant-speed only)
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
					lastUndo.valid = false // AI acted, clear undo
				}
				executeAction(g, g.Players[nonActiveIdx].PlayerID(), action, addLog)
				g.CheckStateBasedActions()
				if g.IsGameOver() {
					return
				}
				continue
			}

			// Both passed in succession
			lastUndo.valid = false
			if g.Stack.IsEmpty() {
				return // move to next step/phase
			}

			// Resolve top of stack
			top := g.Stack.Peek()
			topName := "ability"
			if top.Card != nil {
				topName = top.Card.Name()
			}
			addLog(fmt.Sprintf("Resolving %s", topName))
			g.ResolveTopOfStack()
			g.CheckStateBasedActions()
			if g.IsGameOver() {
				return
			}
			// After resolving, active player gets priority again
		}
	}

	// Main game loop
	for g.Turn <= 100 {
		for _, step := range AllSteps() {
			g.Step = step
			g.Effects.Apply(g)

			if g.IsGameOver() {
				send(PromptNone, nil)
				return
			}

			switch step {
			case Untap:
				g.doUntap()
				addLog(fmt.Sprintf("── Turn %d: %s ──", g.Turn, g.ActivePlayerObj().Name()))

			case Upkeep:
				g.doUpkeep()

			case Draw:
				g.doDraw()
				if g.ActivePlayer == humanIdx {
					addLog("You draw a card")
				} else {
					addLog(fmt.Sprintf("%s draws a card", g.ActivePlayerObj().Name()))
				}

			case PrecombatMain:
				runPriorityLoop(true)

			case BeginCombat:
				// nothing

			case DeclareAttackers:
				activeIdx := g.ActivePlayer
				if activeIdx == humanIdx {
					eligible := getEligibleAttackers(g, g.Players[humanIdx].PlayerID())
					if len(eligible) > 0 {
						send(PromptDeclareAttackers, attackerOptions(eligible))
						action := <-fromTUI
						if action.Type == ActionSelectAttackers {
							if len(action.Attackers) > 0 {
								performAttack(g, action.Attackers, addLog)
							} else {
								addLog("You choose not to attack")
							}
						}
					}
				} else if isAI {
					attackerIDs := aiPlayer.AIAttackers(g)
					if len(attackerIDs) > 0 {
						performAttack(g, attackerIDs, addLog)
					}
				}
				// Priority after attackers declared (if any attackers)
				if len(g.Combat.Groups) > 0 {
					runPriorityLoop(false)
				}

			case DeclareBlockers:
				if len(g.Combat.Groups) == 0 {
					continue
				}
				nonActiveIdx := (g.ActivePlayer + 1) % 2
				if nonActiveIdx == humanIdx {
					eligible := getEligibleBlockers(g, g.Players[humanIdx].PlayerID())
					if len(eligible) > 0 {
						send(PromptDeclareBlockers, blockerOptions(g, eligible))
						action := <-fromTUI
						if action.Type == ActionSelectBlockers && len(action.Blockers) > 0 {
							performBlock(g, action.Blockers, addLog)
						}
					}
				} else if isAI {
					blockers := aiPlayer.AIBlockers(g)
					if len(blockers) > 0 {
						performBlock(g, blockers, addLog)
					}
				}
				// Priority after blockers declared
				if len(g.Combat.Groups) > 0 {
					runPriorityLoop(false)
				}

			case FirstStrikeDamage:
				if g.Combat.HasFirstStrikers(g) {
					g.doCombatDamage(true)
					g.CheckStateBasedActions()
				}

			case CombatDamage:
				if len(g.Combat.Groups) > 0 {
					g.doCombatDamage(false)
					g.CheckStateBasedActions()
					reportCombatResults(g, humanIdx, addLog)
				}

			case EndCombat:
				g.Combat.Reset()

			case PostcombatMain:
				runPriorityLoop(true)

			case EndStep:
				g.FireEvent(GameEvent{
					Type:     EvtEndStep,
					PlayerID: g.ActivePlayerObj().PlayerID(),
				})
				g.PutTriggersOnStack()
				if !g.Stack.IsEmpty() {
					runPriorityLoop(false)
				}

			case Cleanup:
				g.doCleanup()
			}

			g.CheckStateBasedActions()
			if g.IsGameOver() {
				send(PromptNone, nil)
				return
			}
		}

		// Next turn
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

func executeAction(g *Game, playerID uuid.UUID, action PriorityAction, addLog func(string)) {
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
			addLog(fmt.Sprintf("%s casts %s", p.Name(), action.CardName))
		}
	case ActionActivateAbility:
		err = g.ActivateAbilityByIndex(playerID, action.PermanentID, action.AbilityIndex, action.Targets)
		if err == nil {
			addLog(fmt.Sprintf("Activated ability: %s", action.CardName))
		}
	}
	if err != nil {
		addLog(fmt.Sprintf("Error: %v", err))
	}
}

func getEligibleAttackers(g *Game, playerID uuid.UUID) []*Permanent {
	var eligible []*Permanent
	for _, perm := range g.Battlefield {
		if perm.Controller != playerID || !perm.HasType(TypeCreature) {
			continue
		}
		if perm.Tapped {
			continue
		}
		if perm.SummonSick && !perm.HasAbility(Haste) {
			continue
		}
		if !g.Effects.CanAttack(perm.ID()) {
			continue
		}
		if !CanAttackCheck(perm, g) {
			continue
		}
		eligible = append(eligible, perm)
	}
	return eligible
}

func attackerOptions(eligible []*Permanent) []ActionOption {
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

func performAttack(g *Game, attackerIDs []uuid.UUID, addLog func(string)) {
	defender := g.NonActivePlayerObj()
	active := g.ActivePlayerObj()
	for _, id := range attackerIDs {
		atk := g.FindPermanent(id)
		if atk == nil {
			continue
		}
		if !atk.HasAbility(Vigilance) {
			atk.Tapped = true
		}
		g.Combat.AddAttacker(id, defender.PlayerID())
		addLog(fmt.Sprintf("%s attacks with %s %d/%d",
			active.Name(), atk.Name(), atk.CurrentPower(g), atk.CurrentToughness(g)))
		g.FireEvent(GameEvent{
			Type:     EvtDeclaredAttacker,
			SourceID: id,
			PlayerID: active.PlayerID(),
		})
	}
}

func getEligibleBlockers(g *Game, playerID uuid.UUID) []*Permanent {
	var eligible []*Permanent
	for _, perm := range g.Battlefield {
		if perm.Controller != playerID || !perm.HasType(TypeCreature) {
			continue
		}
		if perm.Tapped {
			continue
		}
		eligible = append(eligible, perm)
	}
	return eligible
}

func blockerOptions(g *Game, eligible []*Permanent) []ActionOption {
	var options []ActionOption
	for _, perm := range eligible {
		options = append(options, ActionOption{
			Type:        ActionSelectBlockers,
			Label:       fmt.Sprintf("%s %d/%d", perm.Name(), perm.CurrentPower(g), perm.CurrentToughness(g)),
			PermanentID: perm.ID(),
		})
	}
	// Add "Done" option to finalize blocking
	options = append(options, ActionOption{
		Type:  ActionPass,
		Label: "Done (confirm blocks)",
	})
	return options
}

func performBlock(g *Game, blockers map[uuid.UUID]uuid.UUID, addLog func(string)) {
	nonActive := g.NonActivePlayerObj()
	for blockerID, attackerID := range blockers {
		blocker := g.FindPermanent(blockerID)
		attacker := g.FindPermanent(attackerID)
		if blocker == nil || attacker == nil {
			continue
		}
		if !CanBlock(blocker, attacker, g) {
			continue
		}
		if HasLandwalkEvasion(attacker, nonActive.PlayerID(), g) {
			continue
		}
		g.Combat.AddBlocker(blockerID, attackerID)
		addLog(fmt.Sprintf("%s blocks %s with %s",
			nonActive.Name(), attacker.Name(), blocker.Name()))
		g.FireEvent(GameEvent{
			Type:     EvtDeclaredBlocker,
			SourceID: blockerID,
			TargetID: attackerID,
			PlayerID: nonActive.PlayerID(),
		})
	}
}

func reportCombatResults(g *Game, humanIdx int, addLog func(string)) {
	for _, p := range g.Players {
		addLog(fmt.Sprintf("%s: %d life", p.Name(), p.Life()))
	}
}
