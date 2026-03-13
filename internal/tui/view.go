package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

// frame applies terminal-size constraints so shorter views fully overwrite
// taller previous views (prevents SSH rendering bleed-through).
func (m Model) frame(content string) string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}
	return lipgloss.NewStyle().Width(w).Height(h).Render(content)
}

func (m Model) View() string {
	if m.state == nil {
		return m.frame("\n  Waiting for game to start...\n")
	}

	w := m.width
	if w <= 0 {
		w = 80
	}

	var b strings.Builder

	// ── Title bar ──
	stepName := m.state.Step
	youAreActive := m.state.ActivePlayer == m.state.You.Name
	activeMarker := ""
	switch m.state.Step {
	case "Declare Blockers":
		if youAreActive {
			activeMarker = " (you attack)"
		} else {
			activeMarker = " (declare blockers)"
		}
	case "Declare Attackers":
		if youAreActive {
			activeMarker = " (declare attackers)"
		} else {
			activeMarker = fmt.Sprintf(" (%s attacks)", m.state.ActivePlayer)
		}
	default:
		if youAreActive {
			activeMarker = " (your turn)"
		} else {
			activeMarker = fmt.Sprintf(" (%s's turn)", m.state.ActivePlayer)
		}
	}

	youLife := lifeStr(m.state.You.Life)
	oppLife := lifeStr(m.state.Opponent.Life)

	title := fmt.Sprintf("Turn %d  %s%s     You: %s (lib:%d/gy:%d)   %s: %s (hand:%d, lib:%d/gy:%d)",
		m.state.Turn, stepName, activeMarker, youLife,
		m.state.You.LibraryCount, m.state.You.GraveyardCount,
		m.state.Opponent.Name, oppLife,
		m.state.Opponent.HandCount, m.state.Opponent.LibraryCount, m.state.Opponent.GraveyardCount)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(divider(w))
	b.WriteString("\n")

	// ── Opponent battlefield ──
	b.WriteString(labelStyle.Render(fmt.Sprintf("%s's Battlefield", m.state.Opponent.Name)))
	b.WriteString("\n")
	renderBattlefieldTo(&b, m.state.Opponent.Battlefield, nil, nil)
	b.WriteString(divider(w))
	b.WriteString("\n")

	// During attacker declaration, highlight eligible and selected creatures on your battlefield.
	var youEligible, youSelected map[uuid.UUID]bool
	if m.prompt == interactive.PromptDeclareAttackers {
		youEligible = make(map[uuid.UUID]bool)
		youSelected = make(map[uuid.UUID]bool)
		for i, opt := range m.options {
			youEligible[opt.PermanentID] = true
			if m.selected[i] {
				youSelected[opt.PermanentID] = true
			}
		}
	}

	// ── Your battlefield ──
	b.WriteString(labelStyle.Render("Your Battlefield"))
	b.WriteString("\n")
	renderBattlefieldTo(&b, m.state.You.Battlefield, youEligible, youSelected)
	b.WriteString(divider(w))
	b.WriteString("\n")

	// ── Stack ──
	if len(m.state.StackItems) > 0 {
		b.WriteString(labelStyle.Render("Stack"))
		b.WriteString("\n")
		for i := len(m.state.StackItems) - 1; i >= 0; i-- {
			item := m.state.StackItems[i]
			kind := "spell"
			if item.IsAbility {
				kind = "ability"
			}
			line := fmt.Sprintf("   %s (%s by %s)", item.Name, kind, item.Controller)
			if len(item.Targets) > 0 {
				line += " → " + strings.Join(item.Targets, ", ")
			}
			b.WriteString(line + "\n")
		}
		b.WriteString(divider(w))
		b.WriteString("\n")
	}

	// ── Hand ──
	// Compute which cards are castable so non-castable cards can be dimmed.
	castableIDs := make(map[uuid.UUID]bool)
	hasPriority := m.prompt == interactive.PromptMainPhaseAction || m.prompt == interactive.PromptPriority
	for _, opt := range m.options {
		if opt.Type == interactive.ActionCastSpell || opt.Type == interactive.ActionPlayLand {
			castableIDs[opt.CardID] = true
		}
	}

	b.WriteString(labelStyle.Render(fmt.Sprintf("Hand (%d cards)", len(m.state.You.Hand))))
	b.WriteString("\n")
	if len(m.state.You.Hand) == 0 {
		b.WriteString("   (empty)\n")
	} else {
		b.WriteString("   ")
		for i, c := range m.state.You.Hand {
			if i > 0 {
				b.WriteString("  ")
			}
			isDim := hasPriority && !castableIDs[c.ID]
			if isDim {
				b.WriteString(dimHandCardStyle.Render(c.Name))
				if !c.IsLand && c.ManaCost != "" {
					b.WriteString(" " + dimHandCardStyle.Render(c.ManaCost))
				}
			} else if c.IsLand {
				b.WriteString(landCardStyle.Render(c.Name))
			} else {
				b.WriteString(handCardStyle.Render(c.Name))
				if c.ManaCost != "" {
					b.WriteString(" " + colorManaCost(c.ManaCost))
				}
			}
		}
		b.WriteString("\n")
	}

	// ── Mana pool ──
	mp := m.state.You.ManaPool
	total := mp.White + mp.Blue + mp.Black + mp.Red + mp.Green + mp.Colorless
	if total > 0 {
		b.WriteString("   Mana: ")
		b.WriteString(renderManaPool(mp))
		b.WriteString("\n")
	}

	b.WriteString(divider(w))
	b.WriteString("\n")

	// ── Help screen ──
	if m.helpVisible {
		b.WriteString(labelStyle.Render("Keybindings  (? or esc to close)"))
		b.WriteString("\n")
		b.WriteString(divider(w))
		b.WriteString("\n")
		helpLines := []string{
			"↑/↓  j/k     Navigate menu",
			"enter        Select / confirm action",
			"space        Toggle (attackers, multi-select)",
			"u            Undo last action",
			"i            Inspect cards  (zone browser)",
			"l            Full game log  (scrollable)",
			"?            This help screen",
			"q  ctrl+c    Quit",
			"",
			"In browser/log/help:   esc to close",
		}
		for _, line := range helpLines {
			b.WriteString(menuNormalStyle.Render(line))
			b.WriteString("\n")
		}
		return b.String()
	}

	// ── Full log viewer ──
	if m.logBrowsing {
		b.WriteString(labelStyle.Render("Game Log (↑/↓ to scroll, esc to close)"))
		b.WriteString("\n")
		b.WriteString(divider(w))
		b.WriteString("\n")
		visibleLines := m.height - 6
		if visibleLines < 5 {
			visibleLines = 5
		}
		start := m.logScroll - visibleLines + 1
		if start < 0 {
			start = 0
		}
		end := start + visibleLines
		if end > len(m.log) {
			end = len(m.log)
		}
		for i := start; i < end; i++ {
			line := m.log[i]
			if i == m.logScroll {
				b.WriteString(menuCursorStyle.Render(fmt.Sprintf("  %3d  %s", i+1, line)))
			} else {
				b.WriteString(logStyle.Render(fmt.Sprintf("  %3d  %s", i+1, line)))
			}
			b.WriteString("\n")
		}
		b.WriteString(divider(w))
		b.WriteString("\n")
		b.WriteString(menuNormalStyle.Render(fmt.Sprintf("  line %d / %d", m.logScroll+1, len(m.log))))
		b.WriteString("\n")
		return b.String()
	}

	// ── Card browser overlay ──
	if m.browsing && len(m.browseItems) > 0 {
		b.WriteString(labelStyle.Render("Card Browser (esc to close)"))
		b.WriteString("\n")
		for i, item := range m.browseItems {
			if item.isHeader {
				b.WriteString(labelStyle.Render(item.label))
			} else if i == m.browseCursor {
				b.WriteString(menuCursorStyle.Render(fmt.Sprintf("> %s", item.label)))
			} else {
				b.WriteString(menuNormalStyle.Render(item.label))
			}
			b.WriteString("\n")
		}
		b.WriteString(divider(w))
		b.WriteString("\n")
		// Show detail for selected card
		if m.browseCursor >= 0 && m.browseCursor < len(m.browseItems) && !m.browseItems[m.browseCursor].isHeader {
			b.WriteString(renderCardDetail(m.browseItems[m.browseCursor].detail))
			b.WriteString("\n")
		}
		return b.String()
	}

	// ── Game over or action menu ──
	if m.gameOver {
		result := "Game Over!"
		if m.winner == m.state.You.Name {
			result = "YOU WIN!"
		} else if m.winner != "" {
			result = fmt.Sprintf("%s WINS!", strings.ToUpper(m.winner))
		}
		b.WriteString(gameOverStyle.Render(result))
		b.WriteString("\n  Press q to quit\n")
	} else if m.pendingChoice != nil {
		req := m.pendingChoice
		title, hint := choiceTitle(req)
		b.WriteString(labelStyle.Render(title))
		b.WriteString("\n")
		if hint != "" {
			b.WriteString(menuNormalStyle.Render("  " + hint))
			b.WriteString("\n")
		}
		isMulti := req.Type == interactive.ChoiceCardsFromHand
		for i, opt := range req.Options {
			label := opt.Label
			if isMulti {
				check := "[ ]"
				if m.choiceSelected[i] {
					check = selectedCheckStyle.Render("[x]")
				}
				if i == m.choiceCursor {
					b.WriteString(fmt.Sprintf(" %s %s", check, menuCursorStyle.Render(label)))
				} else {
					b.WriteString(fmt.Sprintf(" %s %s", check, menuNormalStyle.Render(label)))
				}
			} else {
				if i == m.choiceCursor {
					b.WriteString(menuCursorStyle.Render(fmt.Sprintf("> %s", label)))
				} else {
					b.WriteString(menuNormalStyle.Render(label))
				}
			}
			b.WriteString("\n")
		}
		if isMulti {
			b.WriteString("\n")
			b.WriteString(menuNormalStyle.Render(fmt.Sprintf("  Press enter to confirm (%d selected / %d required)", countSelected(m.choiceSelected), req.Amount)))
			b.WriteString("\n")
		}
	} else if m.selectingTarget {
		b.WriteString(labelStyle.Render("Choose Target (esc to cancel)"))
		b.WriteString("\n")
		for i, label := range m.targetLabels {
			if i == m.targetCursor {
				b.WriteString(menuCursorStyle.Render(fmt.Sprintf("> %s", label)))
			} else {
				b.WriteString(menuNormalStyle.Render(label))
			}
			b.WriteString("\n")
		}
	} else if m.assigningBlocker {
		attackers := getAttackers(m.state)
		if len(m.blockerAssignments) > 0 {
			b.WriteString(labelStyle.Render("Blockers assigned:"))
			b.WriteString("\n")
			for _, ba := range m.blockerAssignments {
				blockerName := permanentNameFromState(m.state, ba.BlockerID)
				attackerName := permanentNameFromState(m.state, ba.AttackerID)
				b.WriteString(menuNormalStyle.Render(fmt.Sprintf("%s → %s", blockerName, attackerName)))
				b.WriteString("\n")
			}
		}
		b.WriteString(labelStyle.Render("Assign blocker to which attacker?"))
		b.WriteString("\n")
		for i, atk := range attackers {
			if i == m.blockerAssignCursor {
				b.WriteString(menuCursorStyle.Render(fmt.Sprintf("> %s", atk.Name)))
			} else {
				b.WriteString(menuNormalStyle.Render(atk.Name))
			}
			b.WriteString("\n")
		}
		// Done option
		if m.blockerAssignCursor == len(attackers) {
			b.WriteString(menuCursorStyle.Render("> Done (submit blocks)"))
		} else {
			b.WriteString(menuNormalStyle.Render("Done (submit blocks)"))
		}
		b.WriteString("\n")
	} else if m.prompt != interactive.PromptNone && len(m.options) > 0 {
		promptLabel := "Actions"
		switch m.prompt {
		case interactive.PromptMainPhaseAction:
			promptLabel = "Main Phase Actions"
		case interactive.PromptPriority:
			promptLabel = "Priority (instant speed)"
		case interactive.PromptDeclareAttackers:
			promptLabel = "Select Attackers (space=toggle, enter=confirm)"
		case interactive.PromptDeclareBlockers:
			promptLabel = "Select Blockers (enter=assign, then pick attacker)"
		}
		b.WriteString(labelStyle.Render(promptLabel))
		b.WriteString("\n")
		if m.prompt == interactive.PromptDeclareBlockers && len(m.blockerAssignments) > 0 {
			b.WriteString(labelStyle.Render("  Assigned so far:"))
			b.WriteString("\n")
			for _, ba := range m.blockerAssignments {
				blockerName := permanentNameFromState(m.state, ba.BlockerID)
				attackerName := permanentNameFromState(m.state, ba.AttackerID)
				b.WriteString(menuNormalStyle.Render(fmt.Sprintf("  %s → %s", blockerName, attackerName)))
				b.WriteString("\n")
			}
		}

		isMultiSelect := m.prompt == interactive.PromptDeclareAttackers

		for i, opt := range m.options {
			if isMultiSelect {
				check := "[ ]"
				if m.selected[i] {
					check = selectedCheckStyle.Render("[x]")
				}
				if i == m.cursor {
					b.WriteString(fmt.Sprintf(" %s %s", check, menuCursorStyle.Render(opt.Label)))
				} else {
					b.WriteString(fmt.Sprintf(" %s %s", check, menuNormalStyle.Render(opt.Label)))
				}
			} else {
				if opt.Type == interactive.ActionCastSpell && opt.CardName != "" {
					// Show card name and mana cost separately so the cost can be colored.
					if i == m.cursor {
						b.WriteString(menuCursorStyle.Render(fmt.Sprintf("> Cast %s", opt.CardName)))
					} else {
						b.WriteString(menuNormalStyle.Render(fmt.Sprintf("Cast %s", opt.CardName)))
					}
					if opt.ManaCost != "" {
						b.WriteString(" " + colorManaCost(opt.ManaCost))
					}
				} else {
					if i == m.cursor {
						b.WriteString(menuCursorStyle.Render(fmt.Sprintf("> %s", opt.Label)))
					} else {
						b.WriteString(menuNormalStyle.Render(opt.Label))
					}
				}
			}
			b.WriteString("\n")
		}

		if isMultiSelect {
			b.WriteString("\n")
			b.WriteString(menuNormalStyle.Render("Press enter to confirm attackers"))
			b.WriteString("\n")
		}
		var hints []string
		if m.canUndo {
			hints = append(hints, "(u) Undo")
		}
		hints = append(hints, "(i) Inspect cards")
		hints = append(hints, "(l) Full log")
		hints = append(hints, "(?) Help")
		b.WriteString(undoHintStyle.Render("  " + strings.Join(hints, "  ")))
		b.WriteString("\n")
	}

	// ── Log ──
	b.WriteString(divider(w))
	b.WriteString("\n")
	logLines := m.log
	maxLog := 8
	if len(logLines) > maxLog {
		logLines = logLines[len(logLines)-maxLog:]
	}
	for _, l := range logLines {
		b.WriteString(logStyle.Render("  " + l))
		b.WriteString("\n")
	}

	return b.String()
}

// permCardWidth is the fixed visual width of each permanent card block.
const permCardWidth = 20

func renderBattlefieldTo(b *strings.Builder, perms []interactive.PermanentState, eligible, selected map[uuid.UUID]bool) {
	if len(perms) == 0 {
		b.WriteString("   (empty)\n")
		return
	}

	var lands, nonLands []interactive.PermanentState
	for _, p := range perms {
		if p.IsLand {
			lands = append(lands, p)
		} else {
			nonLands = append(nonLands, p)
		}
	}

	// Lands: compact single line
	if len(lands) > 0 {
		b.WriteString("   ")
		for i, p := range lands {
			if i > 0 {
				b.WriteString("  ")
			}
			b.WriteString(renderLandPerm(p))
		}
		b.WriteString("\n")
	}

	// Non-lands: side-by-side card blocks
	if len(nonLands) > 0 {
		cards := make([]string, len(nonLands))
		for i, p := range nonLands {
			cards[i] = buildPermCard(p, eligible[p.ID], selected[p.ID])
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, cards...)
		for _, line := range strings.Split(row, "\n") {
			b.WriteString("   ")
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
}

// renderLandPerm renders a land as a compact single token.
func renderLandPerm(p interactive.PermanentState) string {
	name := p.Name
	if p.Tapped {
		return tappedPermStyle.Render(name + " (T)")
	}
	return landCardStyle.Render(name)
}

// buildPermCard renders a non-land permanent as a fixed-width multi-line card.
// eligible: can be selected as attacker. selected: chosen to attack this declaration.
func buildPermCard(p interactive.PermanentState, eligible, selected bool) string {
	// Choose colors based on status.
	var nameColor lipgloss.Color
	switch {
	case selected:
		nameColor = colorGreen
	case p.Attacking:
		nameColor = colorRed
	case eligible:
		nameColor = colorWhite
	case p.Tapped:
		nameColor = colorTapped
	case p.SummonSick && p.IsCreature:
		nameColor = colorDim
	default:
		nameColor = colorWhite
	}
	nameStyle := lipgloss.NewStyle().Foreground(nameColor).Bold(p.Attacking || selected)

	// Line 0: name + status badge
	badge := ""
	switch {
	case selected:
		badge = " [✓]"
	case eligible:
		badge = " [ ]"
	case p.Attacking:
		badge = " ▶ATK"
	case p.Tapped:
		badge = " (T)"
	case p.SummonSick && p.IsCreature:
		badge = " ~"
	}
	line0 := nameStyle.Render(p.Name + badge)

	// Line 1: mana cost (dim)
	line1 := lipgloss.NewStyle().Foreground(colorDim).Render(p.ManaCost)

	// Line 2: separator
	line2 := lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", permCardWidth))

	// Line 3: P/T and counters
	var line3 string
	if p.IsCreature {
		ptStyle := lipgloss.NewStyle().Foreground(nameColor).Bold(true)
		line3 = ptStyle.Render(fmt.Sprintf("%d/%d", p.Power, p.Toughness))
		if len(p.Counters) > 0 {
			keys := make([]string, 0, len(p.Counters))
			for ct := range p.Counters {
				keys = append(keys, ct)
			}
			sort.Strings(keys)
			var cparts []string
			for _, ct := range keys {
				cparts = append(cparts, fmt.Sprintf("[%s×%d]", ct, p.Counters[ct]))
			}
			line3 += lipgloss.NewStyle().Foreground(colorBlue).Render(" " + strings.Join(cparts, " "))
		}
	}

	// Line 4: keywords (green italic, truncated to fit)
	var line4 string
	if len(p.Keywords) > 0 {
		kws := strings.Join(p.Keywords, ", ")
		line4 = lipgloss.NewStyle().Foreground(colorGreen).Italic(true).Render(kws)
	}

	text := strings.Join([]string{line0, line1, line2, line3, line4}, "\n")
	return lipgloss.NewStyle().Width(permCardWidth).MarginRight(2).Render(text)
}

func renderManaPool(mp interactive.ManaPoolState) string {
	var parts []string
	if mp.White > 0 {
		parts = append(parts, manaSymbolStyle.Foreground(colorManaW).Render(fmt.Sprintf("W:%d", mp.White)))
	}
	if mp.Blue > 0 {
		parts = append(parts, manaSymbolStyle.Foreground(colorManaU).Render(fmt.Sprintf("U:%d", mp.Blue)))
	}
	if mp.Black > 0 {
		parts = append(parts, manaSymbolStyle.Foreground(colorManaB).Render(fmt.Sprintf("B:%d", mp.Black)))
	}
	if mp.Red > 0 {
		parts = append(parts, manaSymbolStyle.Foreground(colorManaR).Render(fmt.Sprintf("R:%d", mp.Red)))
	}
	if mp.Green > 0 {
		parts = append(parts, manaSymbolStyle.Foreground(colorManaG).Render(fmt.Sprintf("G:%d", mp.Green)))
	}
	if mp.Colorless > 0 {
		parts = append(parts, manaSymbolStyle.Foreground(colorDim).Render(fmt.Sprintf("C:%d", mp.Colorless)))
	}
	return strings.Join(parts, "  ")
}

func colorManaCost(mc string) string {
	if mc == "" {
		return ""
	}
	return manaSymbolStyle.Foreground(colorDim).Render(mc)
}

func lifeStr(life int) string {
	if life <= 5 {
		return lifeRedStyle.Render(fmt.Sprintf("%d", life))
	}
	return lifeGreenStyle.Render(fmt.Sprintf("%d", life))
}

func choiceTitle(req *interactive.ChoiceRequest) (title, hint string) {
	switch req.Type {
	case interactive.ChoicePermanent:
		return fmt.Sprintf("Choose permanent — %s", req.Reason), ""
	case interactive.ChoiceCardsFromHand:
		return fmt.Sprintf("Choose %d card(s) — %s", req.Amount, req.Reason), "(space=toggle, enter=confirm)"
	case interactive.ChoiceManaColor:
		return fmt.Sprintf("Choose mana color — %s", req.Reason), ""
	case interactive.ChoiceCardFromLibrary:
		return fmt.Sprintf("Search library — %s", req.Reason), ""
	case interactive.ChoiceMay:
		return fmt.Sprintf("Optional — %s", req.Reason), ""
	case interactive.ChoiceMode:
		return fmt.Sprintf("Choose mode — %s", req.Reason), ""
	}
	return "Choose", ""
}

func countSelected(sel map[int]bool) int {
	n := 0
	for _, v := range sel {
		if v {
			n++
		}
	}
	return n
}

func renderCardDetail(c cardDetail) string {
	var lines []string

	// Name and mana cost
	header := c.Name
	if c.ManaCost != "" && c.ManaCost != "{0}" {
		header += "  " + c.ManaCost
	}
	lines = append(lines, header)

	// Type line
	typeLine := c.Types
	if c.SubTypes != "" {
		typeLine += " — " + c.SubTypes
	}
	lines = append(lines, typeLine)

	// Separator
	lines = append(lines, strings.Repeat("─", 30))

	// Keywords
	if len(c.Keywords) > 0 {
		lines = append(lines, strings.Join(c.Keywords, ", "))
	}

	// Rules text
	if c.RulesText != "" {
		lines = append(lines, c.RulesText)
	}

	// P/T
	if c.IsCreature {
		lines = append(lines, fmt.Sprintf("%d/%d", c.Power, c.Toughness))
	}

	return cardDetailStyle.Render(strings.Join(lines, "\n"))
}
