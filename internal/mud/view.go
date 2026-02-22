package mud

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mage/mage/internal/worlddata"
)

// panelW is the fixed width of the info panel to the right of the map.
const panelW = 26

// npcColors maps NPC ID to a lipgloss hex color for map rendering.
var npcColors = map[string]string{
	// Original 7
	"elder_maren":       "#22c55e", // green
	"merchant_varro":    "#facc15", // yellow
	"wandering_duelist": "#67e8f9", // cyan
	"scavenger_theron":  "#ef4444", // red
	"the_warden":        "#22c55e", // green
	"sister_vael":       "#d946ef", // magenta
	"baron_sengir":      "#ef4444", // red bold
	// Western Green branch
	"asha":   "#4ade80", // bright green
	"orvyn":  "#86efac", // light green
	"cael":   "#22c55e", // forest green
	"greth":  "#fb923c", // orange (beast tamer)
	// Central South branch
	"pardoned_knight": "#fde68a", // pale gold (white knight)
	"nixx":            "#f87171", // light red
	"vex":             "#a855f7", // purple (dark mage)
	// Eastern Coastal branch
	"lyss": "#67e8f9", // cyan (coast)
	"rhen": "#fb923c", // orange-red (harbor fire/water)
	"mora": "#38bdf8", // sky blue (control)
	"pelth": "#34d399", // teal (blue/green salvager)
	// Far Coast branch
	"yso":        "#818cf8", // indigo (deep mage)
	"fen":        "#60a5fa", // blue (sea mage)
	"sea_pirate": "#f97316", // orange-red (pirate)
	// Deep South branch
	"urath": "#7f1d1d", // dark red (warrior)
	"zara":  "#c026d3", // fuchsia (necromancer)
}

var (
	mudTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#facc15")).Padding(0, 1)
	mudLabelStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#9ca3af")).PaddingLeft(1)
	mudNormalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#e5e7eb")).PaddingLeft(2)
	mudCursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#fbbf24")).Bold(true).PaddingLeft(1)
	mudDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280")).PaddingLeft(2)
	mudGoldStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#fbbf24")).Bold(true)
	mudWinStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")).Bold(true).PaddingLeft(2)
	mudLossStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Bold(true).PaddingLeft(2)
	mudHintStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#60a5fa")).Italic(true).PaddingLeft(2)
	mudDivStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#4b5563"))
)

// wordWrap splits text into lines of at most maxW runes, breaking on spaces.
func wordWrap(text string, maxW int) []string {
	if maxW <= 0 {
		return []string{text}
	}
	var lines []string
	words := strings.Fields(text)
	line := ""
	for _, w := range words {
		if line == "" {
			line = w
		} else if len(line)+1+len(w) <= maxW {
			line += " " + w
		} else {
			lines = append(lines, line)
			line = w
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func mudDivider(w int) string {
	if w <= 0 {
		w = 60
	}
	return mudDivStyle.Render(strings.Repeat("─", w))
}

func (m Model) View() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}

	var content string
	switch m.state {
	case stateStarterChest:
		content = m.viewStarterChest(w)
	case stateExamine:
		content = m.viewExamine(w)
	case stateChallenge:
		content = m.viewChallenge(w)
	case stateLoot:
		content = m.viewLoot(w)
	case stateInventory:
		content = m.viewInventory(w)
	case stateSearch:
		content = m.viewSearch(w)
	default:
		content = m.viewRoom(w)
	}

	// Constrain to terminal dimensions so old content doesn't bleed through
	// when a shorter view replaces a taller one (SSH rendering artifact).
	return lipgloss.NewStyle().Width(w).Height(h).Render(content)
}

// ── Room view (NetHack-style map + info panel) ─────────────────────────────

func (m Model) viewRoom(w int) string {
	mapW := w - panelW
	if mapW < 10 {
		mapW = 10
	}
	mapH := m.height
	if mapH <= 0 {
		mapH = 24
	}

	rooms := worlddata.Rooms()
	npcAvail := func(npcID string) bool {
		return m.world.NPCAvailable(m.fingerprint, npcID)
	}

	grid := WorldMap(rooms, m.roomID, npcAvail)

	// Cap mapW to the actual canvas width so we don't pad beyond the map.
	if len(grid) > 0 && mapW > len(grid[0]) {
		mapW = len(grid[0])
	}

	curRoom, ok := worlddata.RoomByID(m.roomID)
	if !ok {
		return "(unknown room)\n"
	}

	colorize := m.buildColorize(rooms, curRoom.ID, npcAvail)
	mapStr := RenderMap(grid, curRoom, mapW, mapH, colorize)
	mapBox := lipgloss.NewStyle().Width(mapW).Height(mapH).Render(mapStr)

	panelStr := m.viewPanel(panelW)
	panelBox := lipgloss.NewStyle().Width(panelW).Height(mapH).Render(panelStr)

	return lipgloss.JoinHorizontal(lipgloss.Top, mapBox, panelBox)
}

// buildColorize returns a per-cell coloriser function for RenderMap.
// It uses room and NPC context to apply NetHack-style terminal colours.
func (m Model) buildColorize(
	rooms []worlddata.RoomDef,
	currentRoomID string,
	npcAvail func(string) bool,
) func(x, y int, r rune) string {
	cells := buildCellInfo(rooms)

	// Precompute which rooms are locked (for dimming wall tiles).
	lockedRooms := map[string]bool{}
	for _, r := range rooms {
		if r.GoldLock > 0 || r.BossLock != "" {
			lockedRooms[r.ID] = true
		}
	}

	playerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")).Bold(true)
	corridorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4b5563"))

	return func(x, y int, r rune) string {
		if r == ' ' {
			return " "
		}

		// Player takes priority.
		if r == '@' {
			return playerStyle.Render("@")
		}

		ci := cells[[2]int{x, y}]

		// NPC glyph.
		if ci.npcID != "" {
			avail := npcAvail(ci.npcID)
			if avail {
				color, ok := npcColors[ci.npcID]
				if !ok {
					color = "#e5e7eb"
				}
				bold := ci.npcID == "baron_sengir" || ci.npcID == "zara" || ci.npcID == "fen"
				s := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(bold)
				return s.Render(string(r))
			}
			// Defeated / cooling down: dim gray.
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280")).Render(string(r))
		}

		// Room tile.
		if ci.roomID != "" {
			isCurrent := ci.roomID == currentRoomID
			isLocked := lockedRooms[ci.roomID]
			return mapTileStyle(r, isCurrent, isLocked).Render(string(r))
		}

		// Corridor.
		if r == '#' {
			return corridorStyle.Render("#")
		}

		return string(r)
	}
}

// mapTileStyle returns a lipgloss style for a room wall or floor tile.
func mapTileStyle(r rune, isCurrent, isLocked bool) lipgloss.Style {
	switch r {
	case '+', '-', '|':
		if isCurrent {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#e5e7eb")).Bold(true)
		}
		if isLocked {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#374151"))
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#9ca3af"))
	case '.':
		if isCurrent {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#1f2937"))
	}
	return lipgloss.NewStyle()
}

// viewPanel renders the right-hand info panel (room name, NPCs, gold, hints).
func (m Model) viewPanel(w int) string {
	var b strings.Builder

	room, ok := worlddata.RoomByID(m.roomID)
	if !ok {
		return "(unknown)\n"
	}

	// Room name.
	b.WriteString(mudTitleStyle.Render(room.Name))
	b.WriteString("\n")
	b.WriteString(mudDivider(w - 2))
	b.WriteString("\n")

	// Description excerpt (first line, truncated to panel width).
	desc := room.Description
	if idx := strings.Index(desc, "\n"); idx != -1 {
		desc = desc[:idx]
	}
	maxDesc := w - 2
	if len(desc) > maxDesc {
		desc = desc[:maxDesc-1] + "…"
	}
	b.WriteString(mudDimStyle.Render(desc))
	b.WriteString("\n\n")

	// NPCs.
	for npcIdx, id := range room.NPCIDs {
		npcDef, exists := worlddata.NPCByID(id)
		if !exists {
			continue
		}
		g := npcDef.Glyph
		if g == 0 {
			g = '?'
		}
		avail := m.world.NPCAvailable(m.fingerprint, id)

		anteTag := ""
		if npcDef.AnteForced {
			anteTag = "[A]"
		}
		// Truncate name to fit: glyph(1) + space(1) + name + space(1) + tag(3) ≤ w-2
		maxName := w - 2 - 3 // glyph, space, tag area
		if npcDef.AnteForced {
			maxName -= 4
		}
		name := npcDef.Name
		if len(name) > maxName {
			name = name[:maxName-2] + ".."
		}
		entry := fmt.Sprintf("%c %s", g, name)
		if anteTag != "" {
			entry += " " + anteTag
		}

		if avail {
			color, ok := npcColors[id]
			if !ok {
				color = "#e5e7eb"
			}
			s := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).PaddingLeft(1)
			if npcIdx == m.npcCursor {
				s = s.Bold(true)
			}
			b.WriteString(s.Render(entry))
		} else {
			b.WriteString(mudDimStyle.Render(entry + " (rest)"))
		}
		b.WriteString("\n")
	}

	if len(room.NPCIDs) > 0 {
		b.WriteString("\n")
	}

	// Other players in room.
	players := m.world.PlayersInRoom(m.roomID)
	others := make([]string, 0, len(players))
	for _, fp := range players {
		if fp != m.fingerprint {
			others = append(others, fp[:8])
		}
	}
	if len(others) > 0 {
		b.WriteString(mudDimStyle.Render("Also: " + strings.Join(others, ", ")))
		b.WriteString("\n\n")
	}

	// Gold and deck.
	b.WriteString(mudGoldStyle.Render(fmt.Sprintf(" ◈ %d gold", m.coll.Gold)))
	b.WriteString("\n")
	if m.coll.HasDeck() {
		deckName := m.coll.ActiveSavedDeck().Name
		if len(deckName) > w-8 {
			deckName = deckName[:w-9] + ".."
		}
		b.WriteString(mudDimStyle.Render("  Deck: " + deckName))
		b.WriteString("\n")
	}

	// Status / error message — word-wrapped to panel width.
	if m.statusMsg != "" {
		b.WriteString("\n")
		for _, line := range wordWrap(m.statusMsg, w-3) {
			b.WriteString(mudDimStyle.Render(line))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(mudDivider(w - 2))
	b.WriteString("\n")
	b.WriteString(mudHintStyle.Render("nsew/hjkl move"))
	b.WriteString("\n")
	b.WriteString(mudHintStyle.Render("f fight  x examine"))
	b.WriteString("\n")
	b.WriteString(mudHintStyle.Render("t talk   r search"))
	b.WriteString("\n")
	b.WriteString(mudHintStyle.Render("i inv  d deck  q quit"))
	b.WriteString("\n")

	return b.String()
}

// ── Examine view ────────────────────────────────────────────────────────────

func (m Model) viewExamine(w int) string {
	var b strings.Builder
	if m.selectedNPC == nil {
		return "(nothing selected)\n"
	}
	npc := m.selectedNPC
	b.WriteString(mudTitleStyle.Render(npc.Name))
	b.WriteString("\n")
	b.WriteString(mudDivider(w))
	b.WriteString("\n\n")
	for _, line := range strings.Split(npc.Description, "\n") {
		b.WriteString(mudNormalStyle.Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if npc.AnteForced {
		b.WriteString(mudLossStyle.Render("  This opponent always plays with ante."))
		b.WriteString("\n")
	}
	b.WriteString(mudDivider(w))
	b.WriteString("\n")
	b.WriteString(mudHintStyle.Render("f / enter = challenge  esc = back"))
	b.WriteString("\n")
	return b.String()
}

// ── Challenge view ──────────────────────────────────────────────────────────

func (m Model) viewChallenge(w int) string {
	var b strings.Builder
	if m.selectedNPC == nil {
		return "(no target)\n"
	}
	npc := m.selectedNPC
	b.WriteString(mudTitleStyle.Render("Challenge: " + npc.Name))
	b.WriteString("\n")
	b.WriteString(mudDivider(w))
	b.WriteString("\n\n")

	if m.coll.HasDeck() {
		deck := m.coll.ActiveSavedDeck()
		total := 0
		for _, e := range deck.Main {
			total += e.Count
		}
		b.WriteString(mudNormalStyle.Render(fmt.Sprintf("Your deck: %s  (%d cards)", deck.Name, total)))
		b.WriteString("\n\n")
	}

	if npc.AnteForced {
		b.WriteString(mudLossStyle.Render("  Ante is mandatory. One card from each deck will be at stake."))
	} else if m.anteEnabled {
		b.WriteString(mudWinStyle.Render("  Ante: ON  [a to disable]"))
	} else {
		b.WriteString(mudNormalStyle.Render("  Ante: off  [a to enable]"))
	}
	b.WriteString("\n\n")

	b.WriteString(mudDivider(w))
	b.WriteString("\n")
	b.WriteString(mudHintStyle.Render("enter / y = fight  a = toggle ante  n / esc = back"))
	b.WriteString("\n")
	return b.String()
}

// ── Loot view ───────────────────────────────────────────────────────────────

func (m Model) viewLoot(w int) string {
	var b strings.Builder
	if m.loot == nil {
		return "(no loot)\n"
	}
	l := m.loot

	if l.Won {
		b.WriteString(mudWinStyle.Render("  Victory!"))
	} else {
		b.WriteString(mudLossStyle.Render("  Defeat."))
	}
	b.WriteString("\n")
	b.WriteString(mudDivider(w))
	b.WriteString("\n\n")

	if l.AnteCard != "" {
		if l.Won {
			b.WriteString(mudWinStyle.Render(fmt.Sprintf("  Ante claimed: %s", l.AnteCard)))
		} else {
			b.WriteString(mudLossStyle.Render(fmt.Sprintf("  Ante lost: %s", l.AnteCard)))
		}
		b.WriteString("\n")
	}

	if l.Won {
		if l.GoldEarned > 0 {
			b.WriteString(mudGoldStyle.Render(fmt.Sprintf("  +%d gold", l.GoldEarned)))
			b.WriteString("\n")
		}
		if len(l.Cards) > 0 {
			b.WriteString(mudLabelStyle.Render("  Cards earned:"))
			b.WriteString("\n")
			for _, c := range l.Cards {
				b.WriteString(mudWinStyle.Render("    + " + c))
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(mudDivider(w))
	b.WriteString("\n")
	b.WriteString(mudHintStyle.Render("  any key to continue"))
	b.WriteString("\n")
	return b.String()
}

// ── Inventory view ──────────────────────────────────────────────────────────

func (m Model) viewInventory(w int) string {
	var b strings.Builder
	b.WriteString(mudTitleStyle.Render("Collection"))
	b.WriteString("  ")
	b.WriteString(mudGoldStyle.Render(fmt.Sprintf("◈ %d gold", m.coll.Gold)))
	b.WriteString("\n")
	b.WriteString(mudDivider(w))
	b.WriteString("\n\n")

	if m.coll.HasDeck() {
		deck := m.coll.ActiveSavedDeck()
		total := 0
		for _, e := range deck.Main {
			total += e.Count
		}
		b.WriteString(mudNormalStyle.Render(fmt.Sprintf("Active deck: %s  (%d cards)", deck.Name, total)))
		b.WriteString("\n\n")
	}

	// Sorted card list
	type cardEntry struct{ name string; count int }
	var cards []cardEntry
	for name, count := range m.coll.Cards {
		cards = append(cards, cardEntry{name, count})
	}
	sort.Slice(cards, func(i, j int) bool { return cards[i].name < cards[j].name })

	visRows := m.height - 10
	if visRows < 5 {
		visRows = 5
	}
	start := m.invScroll
	if start < 0 {
		start = 0
	}
	end := start + visRows
	if end > len(cards) {
		end = len(cards)
	}

	for _, c := range cards[start:end] {
		b.WriteString(mudNormalStyle.Render(fmt.Sprintf("%-30s ×%d", c.name, c.count)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(mudDimStyle.Render(fmt.Sprintf("  %d unique cards total", len(cards))))
	b.WriteString("\n")
	b.WriteString(mudDivider(w))
	b.WriteString("\n")
	b.WriteString(mudHintStyle.Render("j/k scroll  esc to close"))
	b.WriteString("\n")
	return b.String()
}

// ── Search view ─────────────────────────────────────────────────────────────

func (m Model) viewSearch(w int) string {
	var b strings.Builder
	b.WriteString(mudTitleStyle.Render("Searching..."))
	b.WriteString("\n")
	b.WriteString(mudDivider(w))
	b.WriteString("\n\n")
	for _, line := range m.searchLines {
		if strings.HasPrefix(line, "Found:") {
			b.WriteString(mudWinStyle.Render("  " + line))
		} else {
			b.WriteString(mudNormalStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(mudHintStyle.Render("  any key to continue"))
	b.WriteString("\n")
	return b.String()
}

// ── Starter chest ────────────────────────────────────────────────────────────

func (m Model) viewStarterChest(w int) string {
	var b strings.Builder
	b.WriteString(mudTitleStyle.Render("A weathered chest"))
	b.WriteString("\n")
	b.WriteString(mudDivider(w))
	b.WriteString("\n\n")
	b.WriteString(mudNormalStyle.Render("Inside you find four decks. You may only take one."))
	b.WriteString("\n\n")

	archetypes := worlddata.StarterArchetypes
	for i, arch := range archetypes {
		total := 0
		for _, e := range arch.Deck {
			total += e.Count
		}
		label := fmt.Sprintf("%s  (%d cards)", arch.Name, total)
		if i == m.chestCursor {
			b.WriteString(mudCursorStyle.Render("> " + label))
		} else {
			b.WriteString(mudNormalStyle.Render(label))
		}
		b.WriteString("\n")
		if i == m.chestCursor {
			b.WriteString(mudDimStyle.Render("    " + arch.Description))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(mudDivider(w))
	b.WriteString("\n")
	b.WriteString(mudHintStyle.Render("j/k navigate  enter = take this deck"))
	b.WriteString("\n")
	return b.String()
}
