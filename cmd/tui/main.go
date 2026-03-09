package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mage/mage/internal/tui"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/interactive"
	"github.com/mage/mage/pkg/mage/interactive/ai"

	_ "github.com/mage/mage/cards" // register all card sets
)

type aiPersonality struct {
	Name   string
	Create func(string) *ai.AIPlayer
}

var personalities = []aiPersonality{
	{"Aggro", func(n string) *ai.AIPlayer { return ai.NewWeightedAI(n, ai.AggroWeighted) }},
	{"Control", func(n string) *ai.AIPlayer { return ai.NewWeightedAI(n, ai.ControlWeighted) }},
	{"Midrange", func(n string) *ai.AIPlayer { return ai.NewWeightedAI(n, ai.MidrangeWeighted) }},
	{"Tempo", func(n string) *ai.AIPlayer { return ai.NewWeightedAI(n, ai.TempoWeighted) }},
	{"Burn", func(n string) *ai.AIPlayer { return ai.NewWeightedAI(n, ai.BurnWeighted) }},
}

func main() {
	// Pick your deck
	fmt.Println("Choose your deck:")
	for i, a := range tui.Archetypes {
		fmt.Printf("  %d. %s\n", i+1, a.Name)
	}
	humanDeck := promptChoice("Deck", len(tui.Archetypes))

	// Pick AI deck
	fmt.Println("\nChoose AI deck:")
	for i, a := range tui.Archetypes {
		fmt.Printf("  %d. %s\n", i+1, a.Name)
	}
	aiDeck := promptChoice("Deck", len(tui.Archetypes))

	// Pick AI personality
	fmt.Println("\nChoose AI personality:")
	for i, p := range personalities {
		fmt.Printf("  %d. %s\n", i+1, p.Name)
	}
	aiPers := promptChoice("Personality", len(personalities))

	// Create players
	human := interactive.NewHumanPlayer("You")
	aiPlayer := personalities[aiPers].Create("AI")

	// Build decks
	humanCards := tui.BuildDeck(tui.Archetypes[humanDeck].Entries, human.PlayerID())
	aiCards := tui.BuildDeck(tui.Archetypes[aiDeck].Entries, aiPlayer.PlayerID())

	for _, c := range humanCards {
		human.AddToLibrary(c)
	}
	for _, c := range aiCards {
		aiPlayer.AddToLibrary(c)
	}

	// Create game
	g := mage.NewGame(human, aiPlayer)
	tui.DrawOpeningHand(human)
	tui.DrawOpeningHand(aiPlayer)
	ai.MulliganAI(aiPlayer)

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

func promptChoice(label string, max int) int {
	for {
		fmt.Printf("%s [1-%d]: ", label, max)
		var choice int
		_, err := fmt.Scan(&choice)
		if err == nil && choice >= 1 && choice <= max {
			return choice - 1
		}
		fmt.Println("  Invalid choice, try again.")
	}
}
