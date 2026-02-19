package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mage/mage/pkg/mage"

	// Register all cards
	_ "github.com/mage/mage/cards/limited"
)

func main() {
	// Create players
	human := mage.NewBasePlayer("You")
	ai := mage.NewAIPlayer("AI")

	// Build decks
	humanCards := buildDeck(humanDeck, human.PlayerID())
	aiCards := buildDeck(aiDeck, ai.PlayerID())

	// Load libraries
	for _, c := range humanCards {
		human.AddToLibrary(c)
	}
	for _, c := range aiCards {
		ai.AddToLibrary(c)
	}

	// Create game
	g := mage.NewGame(human, ai)

	// Draw opening hands
	drawOpeningHand(human)
	drawOpeningHand(ai)

	// Channels for communication
	toTUI := make(chan mage.GameMsg, 1)
	fromTUI := make(chan mage.PriorityAction, 1)

	// Start game loop in goroutine
	go mage.RunGameLoop(g, 0, toTUI, fromTUI)

	// Start bubbletea
	model := NewModel(fromTUI, toTUI)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
