package collection

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mage/mage/internal/worlddata"
)

// Builder is a bubbletea model for editing the active deck.
// It renders a two-panel view: collection on the left, current deck on the right.
type Builder struct {
	coll *Collection

	// Sorted collection card names and counts.
	collCards []collCard

	// Which panel is active: "collection" or "deck".
	panel string

	// Which deck tab: "main" or "side".
	tab string

	// Cursor positions within each panel.
	collCursor int
	deckCursor int

	// Filter string for the collection panel.
	filter    string
	filtering bool

	// Validation errors shown after pressing 'v'.
	errors []string

	// Status message shown briefly after an action.
	status string

	width, height int
}

type collCard struct {
	name  string
	owned int
}

var (
	builderTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#facc15")).Padding(0, 1)
	builderLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#9ca3af")).PaddingLeft(1)
	builderCursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#fbbf24")).Bold(true).PaddingLeft(1)
	builderNormalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#e5e7eb")).PaddingLeft(2)
	builderDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280")).PaddingLeft(2)
	builderErrorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).PaddingLeft(2)
	builderOkStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")).PaddingLeft(2)
)

// NewBuilder creates a deckbuilder for the given collection.
func NewBuilder(coll *Collection) Builder {
	b := Builder{
		coll:  coll,
		panel: "collection",
		tab:   "main",
	}
	b.rebuildCollCards()
	return b
}

func (b *Builder) rebuildCollCards() {
	b.collCards = nil
	for name, count := range b.coll.Cards {
		if b.filter != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(b.filter)) {
			continue
		}
		b.collCards = append(b.collCards, collCard{name: name, owned: count})
	}
	sort.Slice(b.collCards, func(i, j int) bool {
		return b.collCards[i].name < b.collCards[j].name
	})
	if b.collCursor >= len(b.collCards) {
		b.collCursor = len(b.collCards) - 1
	}
	if b.collCursor < 0 {
		b.collCursor = 0
	}
}

func (b Builder) Init() tea.Cmd { return nil }

func (b Builder) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height
	case tea.KeyMsg:
		return b.handleKey(msg)
	}
	return b, nil
}

func (b Builder) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	b.status = ""
	b.errors = nil

	if b.filtering {
		switch msg.String() {
		case "esc":
			b.filtering = false
			b.filter = ""
			b.rebuildCollCards()
		case "enter":
			b.filtering = false
			b.rebuildCollCards()
		case "backspace":
			if len(b.filter) > 0 {
				b.filter = b.filter[:len(b.filter)-1]
				b.rebuildCollCards()
			}
		default:
			if len(msg.Runes) > 0 {
				b.filter += string(msg.Runes)
				b.rebuildCollCards()
			}
		}
		return b, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		b.saveActiveDeck()
		return b, tea.Quit

	case "tab":
		// Switch panel
		if b.panel == "collection" {
			b.panel = "deck"
		} else {
			b.panel = "collection"
		}

	case "t":
		// Toggle main/sideboard tab
		if b.tab == "main" {
			b.tab = "side"
		} else {
			b.tab = "main"
		}
		b.deckCursor = 0

	case "up", "k":
		if b.panel == "collection" {
			if b.collCursor > 0 {
				b.collCursor--
			}
		} else {
			if b.deckCursor > 0 {
				b.deckCursor--
			}
		}

	case "down", "j":
		if b.panel == "collection" {
			if b.collCursor < len(b.collCards)-1 {
				b.collCursor++
			}
		} else {
			entries := b.currentDeckEntries()
			if b.deckCursor < len(entries)-1 {
				b.deckCursor++
			}
		}

	case "a", "enter":
		// Add card from collection to deck (if panel == collection)
		// or do nothing if panel == deck.
		if b.panel == "collection" {
			b.addCardToDeck()
		}

	case "r":
		// Remove card from deck (if panel == deck), or remove from collection side.
		if b.panel == "deck" {
			b.removeCardFromDeck()
		} else if b.panel == "collection" {
			b.addCardToDeck() // same as add for now, just mirror
		}

	case "d":
		// Remove selected card from deck when on the deck panel.
		if b.panel == "deck" {
			b.removeCardFromDeck()
		}

	case "/":
		b.filtering = true

	case "v":
		if b.coll.HasDeck() {
			deck := *b.coll.ActiveSavedDeck()
			b.errors = Validate(deck, b.coll)
			if len(b.errors) == 0 {
				b.status = "Deck is valid!"
			}
		}

	case "n":
		// New empty deck
		b.coll.Decks = append(b.coll.Decks, SavedDeck{Name: fmt.Sprintf("Deck %d", len(b.coll.Decks)+1)})
		b.coll.ActiveDeck = len(b.coll.Decks) - 1
		b.status = "New deck created."
	}

	return b, nil
}

func (b *Builder) currentDeckEntries() []worlddata.DeckEntry {
	deck := b.coll.ActiveSavedDeck()
	if deck == nil {
		return nil
	}
	if b.tab == "main" {
		return deck.Main
	}
	return deck.Sideboard
}

func (b *Builder) addCardToDeck() {
	if len(b.collCards) == 0 || b.collCursor >= len(b.collCards) {
		return
	}
	if b.coll.ActiveSavedDeck() == nil {
		b.coll.Decks = append(b.coll.Decks, SavedDeck{Name: "My Deck"})
		b.coll.ActiveDeck = 0
	}
	name := b.collCards[b.collCursor].name
	deck := b.coll.ActiveSavedDeck()

	// Count total copies already in deck.
	total := 0
	for _, e := range deck.Main {
		if e.Name == name {
			total += e.Count
		}
	}
	for _, e := range deck.Sideboard {
		if e.Name == name {
			total += e.Count
		}
	}

	// Check limits.
	if !isBasicLand(name) && total >= MaxCopiesPerCard {
		b.status = fmt.Sprintf("Already have max copies of %s", name)
		return
	}
	if b.coll.Cards[name] <= total {
		b.status = fmt.Sprintf("Not enough copies of %s in collection", name)
		return
	}

	// Add to the active tab.
	if b.tab == "main" {
		for i := range deck.Main {
			if deck.Main[i].Name == name {
				deck.Main[i].Count++
				b.status = fmt.Sprintf("Added %s to main deck", name)
				return
			}
		}
		deck.Main = append(deck.Main, worlddata.DeckEntry{Name: name, Count: 1})
	} else {
		for i := range deck.Sideboard {
			if deck.Sideboard[i].Name == name {
				deck.Sideboard[i].Count++
				b.status = fmt.Sprintf("Added %s to sideboard", name)
				return
			}
		}
		deck.Sideboard = append(deck.Sideboard, worlddata.DeckEntry{Name: name, Count: 1})
	}
	b.status = fmt.Sprintf("Added %s", name)
}

func (b *Builder) removeCardFromDeck() {
	deck := b.coll.ActiveSavedDeck()
	if deck == nil {
		return
	}
	var entries *[]worlddata.DeckEntry
	if b.tab == "main" {
		entries = &deck.Main
	} else {
		entries = &deck.Sideboard
	}
	if b.deckCursor >= len(*entries) {
		return
	}
	name := (*entries)[b.deckCursor].Name
	(*entries)[b.deckCursor].Count--
	if (*entries)[b.deckCursor].Count <= 0 {
		*entries = append((*entries)[:b.deckCursor], (*entries)[b.deckCursor+1:]...)
		if b.deckCursor > 0 && b.deckCursor >= len(*entries) {
			b.deckCursor--
		}
	}
	b.status = fmt.Sprintf("Removed %s", name)
}

func (b *Builder) saveActiveDeck() {
	// Nothing extra to do — the collection struct is modified in place.
	// The caller (SSH handler) will call coll.Save() after the builder quits.
}

func (b Builder) View() string {
	var sb strings.Builder

	w := b.width
	if w <= 0 {
		w = 80
	}

	half := (w - 4) / 2

	sb.WriteString(builderTitleStyle.Render("Deckbuilder"))
	if b.filtering {
		sb.WriteString(builderTitleStyle.Render(fmt.Sprintf("  Filter: %s_", b.filter)))
	} else if b.filter != "" {
		sb.WriteString(builderTitleStyle.Render(fmt.Sprintf("  [filter: %q]", b.filter)))
	}
	sb.WriteString("\n")
	sb.WriteString(builderDimStyle.Render("tab=switch panel  t=main/side  a=add  d=remove  /=filter  v=validate  q=save&quit"))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", w))
	sb.WriteString("\n")

	// Header row
	collHeader := "Collection"
	if b.panel == "collection" {
		collHeader = "> " + collHeader
	}
	deckHeader := fmt.Sprintf("Deck (%s)", b.tab)
	if b.panel == "deck" {
		deckHeader = "> " + deckHeader
	}

	deck := b.coll.ActiveSavedDeck()
	deckCount := 0
	if deck != nil {
		for _, e := range deck.Main {
			deckCount += e.Count
		}
	}

	collLabel := fmt.Sprintf("%-*s", half, collHeader)
	deckLabel := fmt.Sprintf("%-*s", half, fmt.Sprintf("%s  [%d cards]", deckHeader, deckCount))
	sb.WriteString(builderLabelStyle.Render(collLabel))
	sb.WriteString("  ")
	sb.WriteString(builderLabelStyle.Render(deckLabel))
	sb.WriteString("\n")

	// Determine visible window.
	visRows := b.height - 8
	if visRows < 5 {
		visRows = 5
	}

	// Collection side
	collLines := b.renderCollectionPane(visRows, half)
	// Deck side
	deckLines := b.renderDeckPane(visRows, half)

	maxRows := len(collLines)
	if len(deckLines) > maxRows {
		maxRows = len(deckLines)
	}
	for i := 0; i < maxRows; i++ {
		left := ""
		if i < len(collLines) {
			left = collLines[i]
		}
		right := ""
		if i < len(deckLines) {
			right = deckLines[i]
		}
		sb.WriteString(fmt.Sprintf("%-*s  %s\n", half, left, right))
	}

	sb.WriteString(strings.Repeat("─", w))
	sb.WriteString("\n")

	if len(b.errors) > 0 {
		for _, e := range b.errors {
			sb.WriteString(builderErrorStyle.Render("✗ " + e))
			sb.WriteString("\n")
		}
	} else if b.status != "" {
		sb.WriteString(builderOkStyle.Render(b.status))
		sb.WriteString("\n")
	}

	if deck != nil {
		sb.WriteString(builderDimStyle.Render(fmt.Sprintf("Gold: %d", b.coll.Gold)))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (b Builder) renderCollectionPane(rows, width int) []string {
	var lines []string
	start := b.collCursor - rows/2
	if start < 0 {
		start = 0
	}
	end := start + rows
	if end > len(b.collCards) {
		end = len(b.collCards)
	}
	for i := start; i < end; i++ {
		c := b.collCards[i]
		label := fmt.Sprintf("%s ×%d", c.name, c.owned)
		if i == b.collCursor && b.panel == "collection" {
			lines = append(lines, builderCursorStyle.Render(fmt.Sprintf("> %-*s", width-3, label)))
		} else {
			lines = append(lines, builderNormalStyle.Render(fmt.Sprintf("%-*s", width-2, label)))
		}
	}
	return lines
}

func (b Builder) renderDeckPane(rows, width int) []string {
	var lines []string
	entries := b.currentDeckEntries()
	start := b.deckCursor - rows/2
	if start < 0 {
		start = 0
	}
	end := start + rows
	if end > len(entries) {
		end = len(entries)
	}
	for i := start; i < end; i++ {
		e := entries[i]
		label := fmt.Sprintf("%s ×%d", e.Name, e.Count)
		if i == b.deckCursor && b.panel == "deck" {
			lines = append(lines, builderCursorStyle.Render(fmt.Sprintf("> %-*s", width-3, label)))
		} else {
			lines = append(lines, builderNormalStyle.Render(fmt.Sprintf("%-*s", width-2, label)))
		}
	}
	return lines
}
