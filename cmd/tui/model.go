package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/interactive"
)

// gameStateMsg wraps a GameMsg from the game goroutine for bubbletea.
type gameStateMsg interactive.GameMsg

// choiceRequestMsg wraps a ChoiceRequest sent from the game goroutine.
type choiceRequestMsg interactive.ChoiceRequest

// choiceChannelDoneMsg signals the choice channel was closed (game over).
type choiceChannelDoneMsg struct{}

// Model is the bubbletea model for the TUI.
type Model struct {
	toGame   chan<- interactive.PriorityAction
	fromGame <-chan interactive.GameMsg

	// Channels bridging the game goroutine's HumanPlayer choice calls to the TUI.
	fromChoiceReqs <-chan interactive.ChoiceRequest
	toChoiceResps  chan<- interactive.ChoiceResponse

	state    *interactive.GameState
	prompt   interactive.PromptType
	options  []interactive.ActionOption
	cursor   int
	selected map[int]bool // for multi-select (attackers/blockers)
	log      []string
	gameOver bool
	winner   string

	// For blocker assignment
	assigningBlocker    bool
	blockerID           uuid.UUID
	blockerAssignCursor int
	blockerAssignments  []mage.BlockAssignment // accumulated blocker->attacker pairs

	// For target selection
	selectingTarget    bool
	pendingAction      interactive.PriorityAction
	targetOptions      []uuid.UUID
	targetLabels       []string
	targetCursor       int

	// For interactive player choices (sacrifice, discard, mana color, tutor, may)
	pendingChoice   *interactive.ChoiceRequest
	choiceCursor    int
	choiceSelected  map[int]bool // for multi-select (ChoiceCardsFromHand)

	canUndo bool

	// Card browser
	browsing      bool
	browseItems   []browseItem
	browseCursor  int

	// Full log viewer
	logBrowsing bool
	logScroll   int // index of top visible line

	// Help screen
	helpVisible bool

	width, height int
}

// browseItem is one entry in the card browser list.
type browseItem struct {
	label    string // display label in the list
	detail   cardDetail
	isHeader bool // zone header, not selectable
}

// cardDetail holds the info to display for card inspection.
type cardDetail struct {
	Name      string
	ManaCost  string
	Types     string
	SubTypes  string
	Power     int
	Toughness int
	Keywords  []string
	RulesText string
	IsCreature bool
}

// NewModel creates a new TUI model.
func NewModel(toGame chan<- interactive.PriorityAction, fromGame <-chan interactive.GameMsg,
	fromChoiceReqs <-chan interactive.ChoiceRequest, toChoiceResps chan<- interactive.ChoiceResponse) Model {
	return Model{
		toGame:         toGame,
		fromGame:       fromGame,
		fromChoiceReqs: fromChoiceReqs,
		toChoiceResps:  toChoiceResps,
		selected:       make(map[int]bool),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		waitForGameState(m.fromGame),
		waitForChoiceRequest(m.fromChoiceReqs),
	)
}

func waitForGameState(ch <-chan interactive.GameMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return gameStateMsg(interactive.GameMsg{GameOver: true})
		}
		return gameStateMsg(msg)
	}
}

func waitForChoiceRequest(ch <-chan interactive.ChoiceRequest) tea.Cmd {
	return func() tea.Msg {
		req, ok := <-ch
		if !ok {
			return choiceChannelDoneMsg{}
		}
		return choiceRequestMsg(req)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case gameStateMsg:
		gm := interactive.GameMsg(msg)
		m.state = gm.State
		m.prompt = gm.Prompt
		m.options = gm.Options
		m.log = gm.Log
		m.gameOver = gm.GameOver
		m.winner = gm.Winner
		m.cursor = 0
		m.selected = make(map[int]bool)
		m.assigningBlocker = false
		m.selectingTarget = false
		m.pendingChoice = nil
		m.browsing = false
		m.canUndo = gm.CanUndo

		if m.gameOver {
			return m, nil
		}

		// If no prompt, keep listening for next state.
		if m.prompt == interactive.PromptNone {
			return m, waitForGameState(m.fromGame)
		}

		return m, nil

	case choiceRequestMsg:
		req := interactive.ChoiceRequest(msg)
		m.pendingChoice = &req
		m.choiceCursor = 0
		m.choiceSelected = make(map[int]bool)
		return m, nil

	case choiceChannelDoneMsg:
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" || msg.String() == "q" {
		return m, tea.Quit
	}

	if m.gameOver {
		return m, nil
	}

	// Help screen
	if m.helpVisible {
		switch msg.String() {
		case "esc", "?", "q":
			m.helpVisible = false
		}
		return m, nil
	}

	// Full log viewer
	if m.logBrowsing {
		return m.handleLogKey(msg)
	}

	// Card browser mode
	if m.browsing {
		return m.handleBrowseKey(msg)
	}

	// Player choice (sacrifice, discard, mana color, tutor, may) — from game goroutine
	if m.pendingChoice != nil {
		return m.handleChoiceKey(msg)
	}

	// Target selection mode
	if m.selectingTarget {
		return m.handleTargetKey(msg)
	}

	// Blocker assignment mode
	if m.assigningBlocker {
		return m.handleBlockerAssignKey(msg)
	}

	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.options)-1 {
			m.cursor++
		}

	case " ":
		// Toggle selection for multi-select (attackers)
		if m.prompt == interactive.PromptDeclareAttackers {
			m.selected[m.cursor] = !m.selected[m.cursor]
		}

	case "u":
		if m.canUndo {
			m.toGame <- interactive.PriorityAction{Type: interactive.ActionUndo}
			return m, waitForGameState(m.fromGame)
		}

	case "i":
		if m.state != nil {
			m.openBrowser()
		}

	case "l":
		if len(m.log) > 0 {
			m.logBrowsing = true
			m.logScroll = len(m.log) - 1 // start at bottom
		}

	case "?":
		m.helpVisible = true

	case "enter":
		return m.handleEnter()
	}

	return m, nil
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	if len(m.options) == 0 {
		return m, nil
	}

	switch m.prompt {
	case interactive.PromptDeclareAttackers:
		// Collect selected attackers and send
		var attackers []uuid.UUID
		for i, opt := range m.options {
			if m.selected[i] {
				attackers = append(attackers, opt.PermanentID)
			}
		}
		m.toGame <- interactive.PriorityAction{
			Type:      interactive.ActionSelectAttackers,
			Attackers: attackers,
		}
		return m, waitForGameState(m.fromGame)

	case interactive.PromptDeclareBlockers:
		if m.cursor < len(m.options) {
			opt := m.options[m.cursor]
			if opt.Type == interactive.ActionPass {
				// "Done" option — submit all accumulated blockers
				m.toGame <- interactive.PriorityAction{
					Type:     interactive.ActionSelectBlockers,
					Blockers: m.blockerAssignments,
				}
				m.blockerAssignments = nil
				return m, waitForGameState(m.fromGame)
			}
			// Start assigning this blocker to an attacker
			m.assigningBlocker = true
			m.blockerID = opt.PermanentID
			m.blockerAssignCursor = 0
			return m, nil
		}
		return m, nil

	default:
		// Normal action selection
		opt := m.options[m.cursor]

		switch opt.Type {
		case interactive.ActionPass:
			m.toGame <- interactive.PriorityAction{Type: interactive.ActionPass}
			return m, waitForGameState(m.fromGame)

		case interactive.ActionPlayLand:
			m.toGame <- interactive.PriorityAction{
				Type:     interactive.ActionPlayLand,
				CardID:   opt.CardID,
				CardName: opt.Label,
			}
			return m, waitForGameState(m.fromGame)

		case interactive.ActionCastSpell:
			m.pendingAction = interactive.PriorityAction{
				Type:     interactive.ActionCastSpell,
				CardID:   opt.CardID,
				CardName: opt.CardName,
			}

			if opt.NeedsTarget && opt.TargetType != nil && m.state != nil {
				m.selectingTarget = true
				m.targetOptions, m.targetLabels = interactive.GetTargetChoices(m.state, opt)
				m.targetCursor = 0
				if len(m.targetOptions) == 0 {
					m.selectingTarget = false
				}
				return m, nil
			}
			m.toGame <- m.pendingAction
			return m, waitForGameState(m.fromGame)

		case interactive.ActionActivateAbility:
			m.toGame <- interactive.PriorityAction{
				Type:         interactive.ActionActivateAbility,
				PermanentID:  opt.PermanentID,
				AbilityIndex: opt.AbilityIndex,
				CardName:     opt.Label,
			}
			return m, waitForGameState(m.fromGame)
		}
	}

	return m, nil
}

func (m Model) handleChoiceKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	req := m.pendingChoice
	isMulti := req.Type == interactive.ChoiceCardsFromHand

	switch msg.String() {
	case "up", "k":
		if m.choiceCursor > 0 {
			m.choiceCursor--
		}
	case "down", "j":
		if m.choiceCursor < len(req.Options)-1 {
			m.choiceCursor++
		}
	case " ":
		if isMulti {
			m.choiceSelected[m.choiceCursor] = !m.choiceSelected[m.choiceCursor]
		}
	case "esc":
		// Fall back to first option — don't leave the game stuck.
		m.submitChoice(req, 0, nil)
		return m, waitForChoiceRequest(m.fromChoiceReqs)
	case "enter":
		if isMulti {
			var selectedIdx []int
			for i := range req.Options {
				if m.choiceSelected[i] {
					selectedIdx = append(selectedIdx, i)
				}
			}
			// Enforce amount: if not enough selected, auto-fill from top
			if len(selectedIdx) < req.Amount {
				selectedIdx = nil
				for i := 0; i < req.Amount && i < len(req.Options); i++ {
					selectedIdx = append(selectedIdx, i)
				}
			}
			m.submitChoice(req, 0, selectedIdx)
		} else {
			m.submitChoice(req, m.choiceCursor, nil)
		}
		return m, waitForChoiceRequest(m.fromChoiceReqs)
	}
	return m, nil
}

// submitChoice builds and sends a ChoiceResponse for the pending choice.
// singleIdx is used for single-select; multiIdxs is used for multi-select.
func (m *Model) submitChoice(req *interactive.ChoiceRequest, singleIdx int, multiIdxs []int) {
	var resp interactive.ChoiceResponse
	switch req.Type {
	case interactive.ChoicePermanent, interactive.ChoiceCardFromLibrary:
		if singleIdx < len(req.Options) {
			resp.SelectedIDs = []uuid.UUID{req.Options[singleIdx].ID}
		}
	case interactive.ChoiceCardsFromHand:
		for _, idx := range multiIdxs {
			if idx < len(req.Options) {
				resp.SelectedIDs = append(resp.SelectedIDs, req.Options[idx].ID)
			}
		}
	case interactive.ChoiceManaColor:
		if singleIdx < len(req.Options) {
			resp.SelectedColor = req.Options[singleIdx].Color
		}
	case interactive.ChoiceMay:
		resp.Accepted = singleIdx == 0 // option 0 is "Yes"
	case interactive.ChoiceMode:
		resp.SelectedIndex = singleIdx
	}
	m.pendingChoice = nil
	m.choiceSelected = nil
	m.toChoiceResps <- resp
}

func (m Model) handleTargetKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.targetCursor > 0 {
			m.targetCursor--
		}
	case "down", "j":
		if m.targetCursor < len(m.targetOptions)-1 {
			m.targetCursor++
		}
	case "esc":
		m.selectingTarget = false
	case "enter":
		if m.targetCursor < len(m.targetOptions) {
			m.pendingAction.Targets = []uuid.UUID{m.targetOptions[m.targetCursor]}
			m.selectingTarget = false
			m.toGame <- m.pendingAction
			return m, waitForGameState(m.fromGame)
		}
	}
	return m, nil
}

func (m Model) handleBlockerAssignKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Show list of attackers to assign the blocker to
	attackers := getAttackers(m.state)

	switch msg.String() {
	case "up", "k":
		if m.blockerAssignCursor > 0 {
			m.blockerAssignCursor--
		}
	case "down", "j":
		if m.blockerAssignCursor < len(attackers) {
			m.blockerAssignCursor++
		}
	case "esc":
		m.assigningBlocker = false
	case "enter":
		if m.blockerAssignCursor == len(attackers) {
			// "Done" option - submit all blockers
			m.assigningBlocker = false
			m.toGame <- interactive.PriorityAction{
				Type:     interactive.ActionSelectBlockers,
				Blockers: m.blockerAssignments,
			}
			m.blockerAssignments = nil
			return m, waitForGameState(m.fromGame)
		}
		if m.blockerAssignCursor < len(attackers) {
			m.blockerAssignments = append(m.blockerAssignments, mage.BlockAssignment{
				BlockerID:  m.blockerID,
				AttackerID: attackers[m.blockerAssignCursor].ID,
			})
			m.assigningBlocker = false
		}
	}
	return m, nil
}

// openBrowser builds the card browser list from all visible zones.
func (m *Model) openBrowser() {
	var items []browseItem

	// Your Hand
	if len(m.state.You.Hand) > 0 {
		items = append(items, browseItem{label: "── Your Hand ──", isHeader: true})
		for _, c := range m.state.You.Hand {
			label := c.Name
			if c.ManaCost != "" {
				label += "  " + c.ManaCost
			}
			items = append(items, browseItem{
				label: label,
				detail: cardDetail{
					Name:       c.Name,
					ManaCost:   c.ManaCost,
					Types:      c.Types,
					SubTypes:   c.SubTypes,
					Power:      c.Power,
					Toughness:  c.Toughness,
					RulesText:  c.RulesText,
					IsCreature: strings.Contains(c.Types, "Creature"),
				},
			})
		}
	}

	// Your Battlefield
	if len(m.state.You.Battlefield) > 0 {
		items = append(items, browseItem{label: "── Your Battlefield ──", isHeader: true})
		for _, p := range m.state.You.Battlefield {
			label := p.Name
			if p.IsCreature {
				label += fmt.Sprintf(" %d/%d", p.Power, p.Toughness)
			}
			if p.Tapped {
				label += " (T)"
			}
			items = append(items, browseItem{
				label: label,
				detail: cardDetail{
					Name:       p.Name,
					ManaCost:   p.ManaCost,
					Types:      p.Types,
					SubTypes:   p.SubTypes,
					Power:      p.Power,
					Toughness:  p.Toughness,
					Keywords:   p.Keywords,
					RulesText:  p.RulesText,
					IsCreature: p.IsCreature,
				},
			})
		}
	}

	// Opponent Battlefield
	if len(m.state.Opponent.Battlefield) > 0 {
		items = append(items, browseItem{
			label:    fmt.Sprintf("── %s's Battlefield ──", m.state.Opponent.Name),
			isHeader: true,
		})
		for _, p := range m.state.Opponent.Battlefield {
			label := p.Name
			if p.IsCreature {
				label += fmt.Sprintf(" %d/%d", p.Power, p.Toughness)
			}
			if p.Tapped {
				label += " (T)"
			}
			items = append(items, browseItem{
				label: label,
				detail: cardDetail{
					Name:       p.Name,
					ManaCost:   p.ManaCost,
					Types:      p.Types,
					SubTypes:   p.SubTypes,
					Power:      p.Power,
					Toughness:  p.Toughness,
					Keywords:   p.Keywords,
					RulesText:  p.RulesText,
					IsCreature: p.IsCreature,
				},
			})
		}
	}

	// Your Graveyard
	if len(m.state.You.Graveyard) > 0 {
		items = append(items, browseItem{label: "── Your Graveyard ──", isHeader: true})
		for _, c := range m.state.You.Graveyard {
			label := c.Name
			if c.ManaCost != "" {
				label += "  " + c.ManaCost
			}
			items = append(items, browseItem{
				label: label,
				detail: cardDetail{
					Name:       c.Name,
					ManaCost:   c.ManaCost,
					Types:      c.Types,
					SubTypes:   c.SubTypes,
					Power:      c.Power,
					Toughness:  c.Toughness,
					IsCreature: strings.Contains(c.Types, "Creature"),
				},
			})
		}
	}

	// Opponent Graveyard
	if len(m.state.Opponent.Graveyard) > 0 {
		items = append(items, browseItem{
			label:    fmt.Sprintf("── %s's Graveyard ──", m.state.Opponent.Name),
			isHeader: true,
		})
		for _, c := range m.state.Opponent.Graveyard {
			label := c.Name
			if c.ManaCost != "" {
				label += "  " + c.ManaCost
			}
			items = append(items, browseItem{
				label: label,
				detail: cardDetail{
					Name:       c.Name,
					ManaCost:   c.ManaCost,
					Types:      c.Types,
					SubTypes:   c.SubTypes,
					Power:      c.Power,
					Toughness:  c.Toughness,
					IsCreature: strings.Contains(c.Types, "Creature"),
				},
			})
		}
	}

	if len(items) == 0 {
		return
	}

	m.browseItems = items
	m.browseCursor = 0
	// Skip to first non-header item
	for m.browseCursor < len(m.browseItems) && m.browseItems[m.browseCursor].isHeader {
		m.browseCursor++
	}
	m.browsing = true
}

func (m Model) handleLogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "l", "q":
		m.logBrowsing = false
	case "up", "k":
		if m.logScroll > 0 {
			m.logScroll--
		}
	case "down", "j":
		if m.logScroll < len(m.log)-1 {
			m.logScroll++
		}
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleBrowseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "i", "q":
		m.browsing = false
	case "up", "k":
		m.browseCursor--
		// Skip headers
		for m.browseCursor >= 0 && m.browseItems[m.browseCursor].isHeader {
			m.browseCursor--
		}
		if m.browseCursor < 0 {
			// Wrap or clamp — find first non-header
			m.browseCursor = 0
			for m.browseCursor < len(m.browseItems) && m.browseItems[m.browseCursor].isHeader {
				m.browseCursor++
			}
		}
	case "down", "j":
		m.browseCursor++
		// Skip headers
		for m.browseCursor < len(m.browseItems) && m.browseItems[m.browseCursor].isHeader {
			m.browseCursor++
		}
		if m.browseCursor >= len(m.browseItems) {
			// Clamp — find last non-header
			m.browseCursor = len(m.browseItems) - 1
			for m.browseCursor >= 0 && m.browseItems[m.browseCursor].isHeader {
				m.browseCursor--
			}
		}
	}
	return m, nil
}


// permanentNameFromState resolves a permanent ID to its name by searching both battlefields.
func permanentNameFromState(state *interactive.GameState, id uuid.UUID) string {
	for _, p := range state.You.Battlefield {
		if p.ID == id {
			return p.Name
		}
	}
	for _, p := range state.Opponent.Battlefield {
		if p.ID == id {
			return p.Name
		}
	}
	return "?"
}

type attackerInfo struct {
	ID   uuid.UUID
	Name string
}

func getAttackers(state *interactive.GameState) []attackerInfo {
	var attackers []attackerInfo
	for _, p := range state.Opponent.Battlefield {
		if p.Attacking {
			attackers = append(attackers, attackerInfo{ID: p.ID, Name: p.Name})
		}
	}
	return attackers
}
