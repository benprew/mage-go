package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	colorGreen  = lipgloss.Color("#22c55e")
	colorRed    = lipgloss.Color("#ef4444")
	colorBlue   = lipgloss.Color("#60a5fa")
	colorYellow = lipgloss.Color("#facc15")
	colorDim    = lipgloss.Color("#9ca3af")
	colorWhite  = lipgloss.Color("#e5e7eb")
	colorBorder = lipgloss.Color("#4b5563")
	colorTapped = lipgloss.Color("#6b7280")
	colorCursor = lipgloss.Color("#fbbf24")
	colorManaW  = lipgloss.Color("#fef3c7")
	colorManaU  = lipgloss.Color("#93c5fd")
	colorManaB  = lipgloss.Color("#a78bfa")
	colorManaR  = lipgloss.Color("#fca5a5")
	colorManaG  = lipgloss.Color("#86efac")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorYellow).
			Padding(0, 1)

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorDim).
			PaddingLeft(1)

	permStyle = lipgloss.NewStyle().
			Foreground(colorWhite)

	tappedPermStyle = lipgloss.NewStyle().
			Foreground(colorTapped)

	sickPermStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	attackingPermStyle = lipgloss.NewStyle().
				Foreground(colorRed).
				Bold(true)

	handCardStyle = lipgloss.NewStyle().
			Foreground(colorWhite)

	dimHandCardStyle = lipgloss.NewStyle().
			Foreground(colorTapped)

	landCardStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	menuNormalStyle = lipgloss.NewStyle().
			Foreground(colorWhite).
			PaddingLeft(2)

	menuCursorStyle = lipgloss.NewStyle().
			Foreground(colorCursor).
			Bold(true).
			PaddingLeft(1)

	logStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	lifeGreenStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorGreen)

	lifeRedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorRed)

	manaSymbolStyle = lipgloss.NewStyle().Bold(true)

	gameOverStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorYellow).
			Padding(1, 2).
			BorderStyle(lipgloss.DoubleBorder()).
			BorderForeground(colorYellow)

	selectedCheckStyle = lipgloss.NewStyle().
				Foreground(colorGreen).
				Bold(true)

	dividerStyle = lipgloss.NewStyle().
			Foreground(colorBorder)

	undoHintStyle = lipgloss.NewStyle().
			Foreground(colorBlue).
			Italic(true)

	cardDetailStyle = lipgloss.NewStyle().
			Foreground(colorWhite).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)
)

func divider(width int) string {
	if width <= 0 {
		width = 60
	}
	return dividerStyle.Render(strings.Repeat("─", width))
}
