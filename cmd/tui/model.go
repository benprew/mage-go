package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
)

// gameStateMsg wraps a GameMsg from the game goroutine for bubbletea.
type gameStateMsg mage.GameMsg

// Model is the bubbletea model for the TUI.
type Model struct {
	toGame   chan<- mage.PriorityAction
	fromGame <-chan mage.GameMsg

	state    *mage.GameState
	prompt   mage.PromptType
	options  []mage.ActionOption
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
	pendingAction      mage.PriorityAction
	targetOptions      []uuid.UUID
	targetLabels       []string
	targetCursor       int

	canUndo bool

	// Card browser
	browsing      bool
	browseItems   []browseItem
	browseCursor  int

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
func NewModel(toGame chan<- mage.PriorityAction, fromGame <-chan mage.GameMsg) Model {
	return Model{
		toGame:     toGame,
		fromGame:   fromGame,
		selected:   make(map[int]bool),
		blockerAssignments: nil,
	}
}

func (m Model) Init() tea.Cmd {
	return waitForGameState(m.fromGame)
}

func waitForGameState(ch <-chan mage.GameMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return gameStateMsg(mage.GameMsg{GameOver: true})
		}
		return gameStateMsg(msg)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case gameStateMsg:
		gm := mage.GameMsg(msg)
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
		m.browsing = false
		m.canUndo = gm.CanUndo

		if m.gameOver {
			return m, nil
		}

		// If no prompt, just listen for next state
		if m.prompt == mage.PromptNone {
			return m, waitForGameState(m.fromGame)
		}

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

	// Card browser mode
	if m.browsing {
		return m.handleBrowseKey(msg)
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

	case "space":
		// Toggle selection for multi-select (attackers)
		if m.prompt == mage.PromptDeclareAttackers {
			m.selected[m.cursor] = !m.selected[m.cursor]
		}

	case "u":
		if m.canUndo {
			m.toGame <- mage.PriorityAction{Type: mage.ActionUndo}
			return m, waitForGameState(m.fromGame)
		}

	case "i":
		if m.state != nil {
			m.openBrowser()
		}

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
	case mage.PromptDeclareAttackers:
		// Collect selected attackers and send
		var attackers []uuid.UUID
		for i, opt := range m.options {
			if m.selected[i] {
				attackers = append(attackers, opt.PermanentID)
			}
		}
		m.toGame <- mage.PriorityAction{
			Type:      mage.ActionSelectAttackers,
			Attackers: attackers,
		}
		return m, waitForGameState(m.fromGame)

	case mage.PromptDeclareBlockers:
		if m.cursor < len(m.options) {
			opt := m.options[m.cursor]
			if opt.Type == mage.ActionPass {
				// "Done" option — submit all accumulated blockers
				m.toGame <- mage.PriorityAction{
					Type:     mage.ActionSelectBlockers,
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
		case mage.ActionPass:
			m.toGame <- mage.PriorityAction{Type: mage.ActionPass}
			return m, waitForGameState(m.fromGame)

		case mage.ActionPlayLand:
			m.toGame <- mage.PriorityAction{
				Type:     mage.ActionPlayLand,
				CardID:   opt.CardID,
				CardName: opt.Label,
			}
			return m, waitForGameState(m.fromGame)

		case mage.ActionCastSpell:
			if opt.NeedsTarget && opt.TargetType != nil && m.state != nil {
				// Need to select a target first
				m.selectingTarget = true
				m.pendingAction = mage.PriorityAction{
					Type:     mage.ActionCastSpell,
					CardID:   opt.CardID,
					CardName: opt.Label,
				}
				// Get possible targets
				// We need to retrieve them from the game - we'll use the snapshot
				m.targetOptions, m.targetLabels = getTargetChoices(m.state, opt)
				m.targetCursor = 0
				if len(m.targetOptions) == 0 {
					// No valid targets, cancel
					m.selectingTarget = false
				}
				return m, nil
			}
			m.toGame <- mage.PriorityAction{
				Type:     mage.ActionCastSpell,
				CardID:   opt.CardID,
				CardName: opt.Label,
			}
			return m, waitForGameState(m.fromGame)

		case mage.ActionActivateAbility:
			m.toGame <- mage.PriorityAction{
				Type:         mage.ActionActivateAbility,
				PermanentID:  opt.PermanentID,
				AbilityIndex: opt.AbilityIndex,
				CardName:     opt.Label,
			}
			return m, waitForGameState(m.fromGame)
		}
	}

	return m, nil
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
			m.toGame <- mage.PriorityAction{
				Type:     mage.ActionSelectBlockers,
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

// getTargetChoices builds target choices from the game state snapshot.
func getTargetChoices(state *mage.GameState, opt mage.ActionOption) ([]uuid.UUID, []string) {
	var ids []uuid.UUID
	var labels []string

	// Include all creatures from both sides
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

	// Also include players as targets for "any target" spells
	// We'll use a nil UUID convention - the game loop will need to resolve
	// For simplicity, just show creatures. The opponent player UUID isn't in the snapshot.
	// We'll add a "Target opponent" option.
	// We don't have the player UUIDs in the snapshot, but we can use a sentinel.
	ids = append(ids, uuid.Nil) // sentinel for "target opponent"
	labels = append(labels, fmt.Sprintf("Target %s (player)", state.Opponent.Name))

	return ids, labels
}

type attackerInfo struct {
	ID   uuid.UUID
	Name string
}

func getAttackers(state *mage.GameState) []attackerInfo {
	var attackers []attackerInfo
	for _, p := range state.Opponent.Battlefield {
		if p.Attacking {
			attackers = append(attackers, attackerInfo{ID: p.ID, Name: p.Name})
		}
	}
	return attackers
}
