package main

import (
	"fmt"
	"strings"

	"github.com/mage/mage"
)

func (m Model) View() string {
	if m.state == nil {
		return "\n  Waiting for game to start...\n"
	}

	w := m.width
	if w <= 0 {
		w = 80
	}

	var b strings.Builder

	// ── Title bar ──
	stepName := m.state.Step
	activeMarker := ""
	if m.state.ActivePlayer == m.state.You.Name {
		activeMarker = " (your turn)"
	} else {
		activeMarker = fmt.Sprintf(" (%s's turn)", m.state.ActivePlayer)
	}

	youLife := lifeStr(m.state.You.Life)
	oppLife := lifeStr(m.state.Opponent.Life)

	title := fmt.Sprintf("Turn %d  %s%s     You: %s   %s: %s",
		m.state.Turn, stepName, activeMarker, youLife, m.state.Opponent.Name, oppLife)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(divider(w))
	b.WriteString("\n")

	// ── Opponent battlefield ──
	b.WriteString(labelStyle.Render(fmt.Sprintf("%s's Battlefield", m.state.Opponent.Name)))
	b.WriteString("\n")
	renderBattlefieldTo(&b, m.state.Opponent.Battlefield)
	b.WriteString(divider(w))
	b.WriteString("\n")

	// ── Your battlefield ──
	b.WriteString(labelStyle.Render("Your Battlefield"))
	b.WriteString("\n")
	renderBattlefieldTo(&b, m.state.You.Battlefield)
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
			b.WriteString(fmt.Sprintf("   %s (%s by %s)\n", item.Name, kind, item.Controller))
		}
		b.WriteString(divider(w))
		b.WriteString("\n")
	}

	// ── Hand ──
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
			if c.IsLand {
				b.WriteString(landCardStyle.Render(c.Name))
			} else {
				b.WriteString(handCardStyle.Render(fmt.Sprintf("%s %s", c.Name, colorManaCost(c.ManaCost))))
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
	} else if m.prompt != mage.PromptNone && len(m.options) > 0 {
		promptLabel := "Actions"
		switch m.prompt {
		case mage.PromptMainPhaseAction:
			promptLabel = "Main Phase Actions"
		case mage.PromptPriority:
			promptLabel = "Priority (instant speed)"
		case mage.PromptDeclareAttackers:
			promptLabel = "Select Attackers (space=toggle, enter=confirm)"
		case mage.PromptDeclareBlockers:
			promptLabel = "Select Blockers (enter=assign, then pick attacker)"
		}
		b.WriteString(labelStyle.Render(promptLabel))
		b.WriteString("\n")

		isMultiSelect := m.prompt == mage.PromptDeclareAttackers

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
				if i == m.cursor {
					b.WriteString(menuCursorStyle.Render(fmt.Sprintf("> %s", opt.Label)))
				} else {
					b.WriteString(menuNormalStyle.Render(opt.Label))
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
		hints = append(hints, "(i) Inspect card")
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

func renderBattlefieldTo(b *strings.Builder, perms []mage.PermanentState) {
	if len(perms) == 0 {
		b.WriteString("   (empty)\n")
		return
	}

	// Separate lands from non-lands
	var lands, creatures, other []string
	for _, p := range perms {
		s := renderPerm(p)
		if p.IsLand {
			lands = append(lands, s)
		} else if p.IsCreature {
			creatures = append(creatures, s)
		} else {
			other = append(other, s)
		}
	}

	if len(lands) > 0 {
		b.WriteString("   ")
		b.WriteString(strings.Join(lands, "  "))
		b.WriteString("\n")
	}
	if len(creatures) > 0 {
		b.WriteString("   ")
		b.WriteString(strings.Join(creatures, "  "))
		b.WriteString("\n")
	}
	if len(other) > 0 {
		b.WriteString("   ")
		b.WriteString(strings.Join(other, "  "))
		b.WriteString("\n")
	}
}

func renderPerm(p mage.PermanentState) string {
	var parts []string
	parts = append(parts, p.Name)

	if p.Tapped {
		parts = append(parts, "(T)")
	}
	if p.IsCreature {
		parts = append(parts, fmt.Sprintf("%d/%d", p.Power, p.Toughness))
	}
	if p.Attacking {
		parts = append(parts, "ATK")
	}
	for ct, n := range p.Counters {
		parts = append(parts, fmt.Sprintf("[%s x%d]", ct, n))
	}

	text := strings.Join(parts, " ")

	if p.Attacking {
		return attackingPermStyle.Render(text)
	}
	if p.Tapped {
		return tappedPermStyle.Render(text)
	}
	if p.SummonSick && p.IsCreature {
		return sickPermStyle.Render(text)
	}
	return permStyle.Render(text)
}

func renderManaPool(mp mage.ManaPoolState) string {
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
