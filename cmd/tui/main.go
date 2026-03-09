package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mage/mage/internal/tui"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/interactive"

	_ "github.com/mage/mage/cards" // register all card sets
)

func main() {
	// Create players
	human := interactive.NewHumanPlayer("You")
	ai := interactive.NewAIPlayer("AI")

	// Build decks
	humanCards := tui.BuildDeck(tui.Archetypes[0].Entries, human.PlayerID())
	aiCards := tui.BuildDeck(tui.Archetypes[1].Entries, ai.PlayerID())

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
	tui.DrawOpeningHand(human)
	tui.DrawOpeningHand(ai)

	// Start game loop in goroutine
	const aiPause = 400 * time.Millisecond
	go interactive.RunGameLoop(g, 0, aiPause)

	// Start bubbletea
	model := tui.NewModel(human.FromTUI(), human.ToTUI(), human.ChoiceRequests(), human.ChoiceResponses())
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
