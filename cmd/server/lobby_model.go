package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mage/mage/internal/tui"
)

type lobbyState int

const (
	lobbyBrowsing         lobbyState = iota
	lobbySelectDeck                  // choosing a deck archetype
	lobbySelectPersonality           // choosing AI personality (AI games only)
	lobbySelectMode                  // choosing AI mode (heuristic/search/adaptive)
	lobbyWaiting                     // waiting for an opponent to join (pvp create)
)

// matchReadyMsg is sent when a pvp opponent joins the slot.
type matchReadyMsg struct{}

// refreshSlotsMsg triggers a slot list refresh.
type refreshSlotsMsg struct{}

// lobbyModel is the bubbletea model for the SSH lobby.
type lobbyModel struct {
	lobby    *Lobby
	username string
	state    lobbyState

	// Browsing state
	slots      []SlotInfo
	cursor     int // into: slots... | "new game" | "vs AI"
	err        string

	// Deck selection state
	archetypes []tui.DeckArchetype
	deckCursor int
	isAI       bool   // true if "A" was chosen
	joinSlotID string // non-empty if joining an existing slot

	// AI personality + mode selection state
	personalityNames  []string
	personalityCursor int
	modeNames         []string
	modeCursor        int

	// Waiting state
	slot *GameSlot // the created slot we're waiting in

	// Result — set when game is ready; the SSH handler reads this.
	session *PlayerSession
}

func newLobbyModel(lobby *Lobby, username string) lobbyModel {
	return lobbyModel{
		lobby:            lobby,
		username:         username,
		state:            lobbyBrowsing,
		archetypes:       tui.Archetypes,
		slots:            lobby.ListSlots(),
		personalityNames: []string{"Aggro", "Control", "Midrange", "Tempo", "Burn"},
		modeNames:        []string{"Heuristic (fast)", "Search (minimax)", "Adaptive (auto-switch)"},
	}
}

func (m lobbyModel) Init() tea.Cmd {
	return tickRefreshCmd()
}

func tickRefreshCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(_ time.Time) tea.Msg {
		return refreshSlotsMsg{}
	})
}

func waitForMatchCmd(ready <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		<-ready
		return matchReadyMsg{}
	}
}

func (m lobbyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case refreshSlotsMsg:
		if m.state == lobbyBrowsing {
			m.slots = m.lobby.ListSlots()
			// Clamp cursor
			total := len(m.slots) + 2 // +2 for "new game" and "vs AI"
			if m.cursor >= total {
				m.cursor = total - 1
			}
		}
		return m, tickRefreshCmd()

	case matchReadyMsg:
		// Opponent joined — game already started in startPvPGame goroutine.
		// session was set before we transitioned to waiting.
		return m, tea.Quit
	}

	return m, nil
}

// totalBrowseOptions returns the number of entries in the browse list:
// open slots + "new game vs human" + "play vs AI".
func (m lobbyModel) totalBrowseOptions() int {
	return len(m.slots) + 2
}

func (m lobbyModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case lobbyBrowsing:
		return m.handleBrowsingKey(msg)
	case lobbySelectDeck:
		return m.handleDeckSelectKey(msg)
	case lobbySelectPersonality:
		return m.handlePersonalitySelectKey(msg)
	case lobbySelectMode:
		return m.handleModeSelectKey(msg)
	case lobbyWaiting:
		return m.handleWaitingKey(msg)
	}
	return m, nil
}

func (m lobbyModel) handleBrowsingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	total := m.totalBrowseOptions()
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < total-1 {
			m.cursor++
		}

	case "n":
		// Create a new pvp slot
		m.isAI = false
		m.joinSlotID = ""
		m.state = lobbySelectDeck
		m.deckCursor = 0

	case "a":
		// Play vs AI
		m.isAI = true
		m.joinSlotID = ""
		m.state = lobbySelectDeck
		m.deckCursor = 0

	case "enter":
		if m.cursor < len(m.slots) {
			// Join an existing slot
			m.joinSlotID = m.slots[m.cursor].ID
			m.isAI = false
			m.state = lobbySelectDeck
			m.deckCursor = 0
		} else {
			// "New game" or "Play vs AI" row
			offset := m.cursor - len(m.slots)
			switch offset {
			case 0:
				m.isAI = false
				m.joinSlotID = ""
				m.state = lobbySelectDeck
				m.deckCursor = 0
			case 1:
				m.isAI = true
				m.joinSlotID = ""
				m.state = lobbySelectDeck
				m.deckCursor = 0
			}
		}
	}
	return m, nil
}

func (m lobbyModel) handleDeckSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "esc":
		// Back to browsing
		m.state = lobbyBrowsing
		m.slots = m.lobby.ListSlots()

	case "up", "k":
		if m.deckCursor > 0 {
			m.deckCursor--
		}

	case "down", "j":
		if m.deckCursor < len(m.archetypes)-1 {
			m.deckCursor++
		}

	case "enter":
		if m.deckCursor >= len(m.archetypes) {
			return m, nil
		}
		entries := m.archetypes[m.deckCursor].Entries
		sess := newPlayerSession(m.username, entries)

		if m.isAI {
			// Transition to personality selection.
			m.session = sess
			m.state = lobbySelectPersonality
			m.personalityCursor = 2 // default to Midrange
			return m, nil
		}

		if m.joinSlotID != "" {
			// Join existing pvp slot
			_, err := m.lobby.JoinSlot(m.joinSlotID, sess)
			if err != nil {
				m.err = err.Error()
				m.state = lobbyBrowsing
				m.slots = m.lobby.ListSlots()
				return m, nil
			}
			m.session = sess
			return m, tea.Quit
		}

		// Create a new pvp slot and wait
		slot := m.lobby.CreateSlot(sess)
		m.slot = slot
		m.session = sess
		m.state = lobbyWaiting
		return m, waitForMatchCmd(slot.ready)
	}
	return m, nil
}

func (m lobbyModel) handlePersonalitySelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "esc":
		m.state = lobbySelectDeck

	case "up", "k":
		if m.personalityCursor > 0 {
			m.personalityCursor--
		}

	case "down", "j":
		if m.personalityCursor < len(m.personalityNames)-1 {
			m.personalityCursor++
		}

	case "enter":
		if m.personalityCursor >= len(m.personalityNames) {
			return m, nil
		}
		// Proceed to mode selection.
		m.state = lobbySelectMode
		m.modeCursor = 0
		return m, nil
	}
	return m, nil
}

func (m lobbyModel) handleModeSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "esc":
		m.state = lobbySelectPersonality

	case "up", "k":
		if m.modeCursor > 0 {
			m.modeCursor--
		}

	case "down", "j":
		if m.modeCursor < len(m.modeNames)-1 {
			m.modeCursor++
		}

	case "enter":
		if m.modeCursor >= len(m.modeNames) {
			return m, nil
		}
		personality := m.personalityNames[m.personalityCursor]
		// Map mode display names to internal mode names.
		modeMap := map[int]string{0: "Heuristic", 1: "Search", 2: "Adaptive"}
		mode := modeMap[m.modeCursor]
		m.lobby.StartAIGame(m.session, personality, mode)
		return m, tea.Quit
	}
	return m, nil
}

func (m lobbyModel) handleWaitingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		// Cancel the slot and exit
		if m.slot != nil {
			m.lobby.CancelSlot(m.slot)
			m.slot = nil
			m.session = nil
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m lobbyModel) View() string {
	var b strings.Builder
	b.WriteString(" ╔══════════════════════════════╗\n")
	b.WriteString(" ║  mage-go  SSH  Lobby          ║\n")
	b.WriteString(" ╚══════════════════════════════╝\n\n")

	switch m.state {
	case lobbyBrowsing:
		m.renderBrowsing(&b)
	case lobbySelectDeck:
		m.renderDeckSelect(&b)
	case lobbySelectPersonality:
		m.renderPersonalitySelect(&b)
	case lobbySelectMode:
		m.renderModeSelect(&b)
	case lobbyWaiting:
		m.renderWaiting(&b)
	}

	return b.String()
}

func (m lobbyModel) renderBrowsing(b *strings.Builder) {
	if m.err != "" {
		b.WriteString(fmt.Sprintf("  Error: %s\n\n", m.err))
	}

	if len(m.slots) == 0 {
		b.WriteString("  No open games.\n\n")
	} else {
		b.WriteString("  Open games:\n")
		for i, s := range m.slots {
			line := fmt.Sprintf("  Game %s  (%s waiting)", s.ID, s.Name)
			if i == m.cursor {
				b.WriteString(fmt.Sprintf("> %s\n", line))
			} else {
				b.WriteString(fmt.Sprintf("  %s\n", line))
			}
		}
		b.WriteString("\n")
	}

	newIdx := len(m.slots)
	aiIdx := len(m.slots) + 1

	if m.cursor == newIdx {
		b.WriteString("> [N] New game vs human\n")
	} else {
		b.WriteString("  [N] New game vs human\n")
	}
	if m.cursor == aiIdx {
		b.WriteString("> [A] Play vs AI\n")
	} else {
		b.WriteString("  [A] Play vs AI\n")
	}

	b.WriteString("\n  ↑/↓ to move  enter to select  n/a shortcuts  q to quit\n")
}

func (m lobbyModel) renderDeckSelect(b *strings.Builder) {
	b.WriteString("  Choose your deck:\n\n")
	for i, arch := range m.archetypes {
		if i == m.deckCursor {
			b.WriteString(fmt.Sprintf("> %s\n", arch.Name))
		} else {
			b.WriteString(fmt.Sprintf("  %s\n", arch.Name))
		}
	}
	b.WriteString("\n  ↑/↓ navigate  enter to confirm  esc to go back\n")
}

func (m lobbyModel) renderPersonalitySelect(b *strings.Builder) {
	b.WriteString("  Choose AI personality:\n\n")
	for i, name := range m.personalityNames {
		if i == m.personalityCursor {
			b.WriteString(fmt.Sprintf("> %s\n", name))
		} else {
			b.WriteString(fmt.Sprintf("  %s\n", name))
		}
	}
	b.WriteString("\n  ↑/↓ navigate  enter to confirm  esc to go back\n")
}

func (m lobbyModel) renderModeSelect(b *strings.Builder) {
	b.WriteString(fmt.Sprintf("  AI: %s — Choose mode:\n\n", m.personalityNames[m.personalityCursor]))
	for i, name := range m.modeNames {
		if i == m.modeCursor {
			b.WriteString(fmt.Sprintf("> %s\n", name))
		} else {
			b.WriteString(fmt.Sprintf("  %s\n", name))
		}
	}
	b.WriteString("\n  ↑/↓ navigate  enter to confirm  esc to go back\n")
}

func (m lobbyModel) renderWaiting(b *strings.Builder) {
	b.WriteString("  Waiting for an opponent to join...\n\n")
	b.WriteString("  Share your game slot with a friend!\n")
	b.WriteString("  They can join when they connect and see your game in the list.\n\n")
	b.WriteString("  esc or q to cancel\n")
}
