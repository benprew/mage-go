package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"git.sr.ht/~cdcarter/mage-go/internal/tui"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/ai"

	_ "git.sr.ht/~cdcarter/mage-go/cards" // register all card sets
)

type aiPersonality struct {
	Name string
	WP   ai.WeightedPersonality
}

var personalities = []aiPersonality{
	{"Aggro", ai.AggroWeighted},
	{"Control", ai.ControlWeighted},
	{"Midrange", ai.MidrangeWeighted},
	{"Tempo", ai.TempoWeighted},
	{"Burn", ai.BurnWeighted},
}

type aiMode struct {
	Name string
}

var modes = []aiMode{
	{"Heuristic (fast)"},
	{"Minimax Search (stronger, slower)"},
	{"Adaptive (switches aggro/control)"},
}

func createAI(name string, persIdx, modeIdx int) *ai.AIPlayer {
	wp := personalities[persIdx].WP
	switch modeIdx {
	case 1: // Search
		return ai.NewSearchAI(name, ai.DefaultSearchConfig(), wp)
	case 2: // Adaptive
		return ai.NewAdaptiveAI(name)
	default: // Heuristic
		return ai.NewWeightedAI(name, wp)
	}
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

	// Pick AI mode
	fmt.Println("\nChoose AI mode:")
	for i, m := range modes {
		fmt.Printf("  %d. %s\n", i+1, m.Name)
	}
	aiMode := promptChoice("Mode", len(modes))

	// Create players
	human := interactive.NewHumanPlayer("You")
	aiPlayer := createAI("AI", aiPers, aiMode)

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
