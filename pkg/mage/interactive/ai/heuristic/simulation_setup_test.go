package heuristic

import (
	"slices"
	"sync"
	"testing"
	"uuid"

	"github.com/benprew/mage-go/internal/tui"
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
)

type mirrorCheckingStrategy struct {
	t         *testing.T
	once      sync.Once
	onOpening func(mage.Player)
}

func (s *mirrorCheckingStrategy) PriorityAction(p mage.Player, g *mage.Game, _ int, _ bool) interactive.PriorityAction {
	s.once.Do(func() {
		if s.onOpening != nil {
			s.onOpening(p)
		}
		deckNames := func(p mage.Player) []string {
			var names []string
			for _, cards := range [][]*mage.Card{p.Hand(), p.Library(), p.Graveyard()} {
				for _, card := range cards {
					names = append(names, card.Name())
				}
			}
			slices.Sort(names)
			return names
		}
		if !slices.Equal(deckNames(p), deckNames(g.GetOpponent(p.PlayerID()))) {
			s.t.Error("mirror match must give both players the same deck")
		}
	})
	return interactive.PriorityAction{Type: interactive.ActionPass}
}

func simulationCardNames(cards []*mage.Card) []string {
	names := make([]string, len(cards))
	for i, card := range cards {
		names[i] = card.Name()
	}
	return names
}

func TestSimulation_DefaultGameCount(t *testing.T) {
	t.Setenv("SIM_GAMES", "")
	if got := getSimulationGameCount(); got != 30000 {
		t.Fatalf("default comparison must run 30000 games, got %d", got)
	}
	t.Setenv("SIM_GAMES", "10")
	if got := getSimulationGameCount(); got != 10 {
		t.Fatalf("explicit quick comparison must run 10 games, got %d", got)
	}
}

func TestSimulation_PairsReuseOpeningState(t *testing.T) {
	t.Setenv("SIM_GAMES", "2")
	type opening struct{ hand, library []string }
	states := make(map[string][]opening)
	var mu sync.Mutex
	factory := func(ai.WeightedPersonality) ai.AIStrategy {
		return &mirrorCheckingStrategy{t: t, onOpening: func(p mage.Player) {
			mu.Lock()
			defer mu.Unlock()
			states[p.Name()] = append(states[p.Name()], opening{simulationCardNames(p.Hand()), simulationCardNames(p.Library())})
		}}
	}
	run1kGames(t, "Paired openings", factory, factory)
	for _, seat := range []string{"PlayerA", "PlayerB"} {
		got := states[seat]
		if len(got) != 2 || !slices.Equal(got[0].hand, got[1].hand) || !slices.Equal(got[0].library, got[1].library) {
			t.Fatalf("seat %s must reuse its hand and library order", seat)
		}
	}
}

func TestSimulation_MulligansKeepCardOwnership(t *testing.T) {
	player := ai.NewAIPlayer("Player", New(ai.MidrangeWeighted))
	prepareSimulationPlayer(player, []tui.DeckEntry{{Name: "Plains", Count: 20}}, 42)
	if len(player.Hand()) != 5 || len(player.Library()) != 15 {
		t.Fatalf("expected two mulligans with all lands, got hand %d and library %d", len(player.Hand()), len(player.Library()))
	}
	for _, cards := range [][]*mage.Card{player.Hand(), player.Library()} {
		for _, card := range cards {
			if card.Owner() != player.PlayerID() {
				t.Fatal("card owner changed during setup")
			}
		}
	}
}

func TestSimulation_SeparatesDrawsAndTurnLimits(t *testing.T) {
	strategy := &mirrorCheckingStrategy{t: t}
	deck := []tui.DeckEntry{{Name: "Plains", Count: 100}}
	result := playSingleGame(strategy, &mirrorCheckingStrategy{t: t}, deck, deck, 51, 42, 43)
	if !result.turnLimited || !result.unresolvedAtReference || result.winner != 0 {
		t.Fatalf("expected a game still unfinished past turn 50: %+v", result)
	}
	var tally simulationTally
	tally.record(result, true)
	tally.record(simulationGameResult{}, true)
	if tally.draws != 1 || tally.turnLimited != 1 || tally.unresolvedAtReference != 1 {
		t.Fatalf("actual draws and turn limits must be distinct: %+v", tally)
	}
}

func TestSimulation_PrimaryAndStressDecks(t *testing.T) {
	t.Setenv("SIM_INCLUDE_PRIESTESS", "0")
	primary := simulationArchetypes(t)
	if len(primary) != 5 || primary[4].Name != "Merfolk Shaman" {
		t.Fatalf("primary suite must replace Priestess with Merfolk Shaman: %v", primary)
	}
	for _, deck := range primary {
		for _, entry := range deck.Entries {
			if _, err := mage.CreateCard(entry.Name); err != nil {
				t.Fatalf("unavailable card in %s: %v", deck.Name, err)
			}
		}
	}
	t.Setenv("SIM_INCLUDE_PRIESTESS", "1")
	stress := simulationArchetypes(t)
	if len(stress) != 6 || stress[4].Name != "Priestess (W walls)" {
		t.Fatal("stress suite must retain Priestess")
	}
	if len(tui.Archetypes) != 5 || tui.Archetypes[4].Name != "Priestess (W walls)" {
		t.Fatal("benchmark setup changed the lobby decks")
	}
}

func TestSimulationPairedInterval(t *testing.T) {
	var pairs [3][3]int64
	pairs[1][1] = 280
	pairs[2][0] = 20
	lower, upper, ok := simulationPairedInterval(320, 280, pairs)
	if !ok || lower <= 50 || lower >= 100*320.0/600 || upper <= 100*320.0/600 {
		t.Fatalf("paired wins must retain their reduced variance: %v %v %v", lower, upper, ok)
	}
	pairs = [3][3]int64{}
	pairs[2][0], pairs[0][2], pairs[1][1] = 100, 100, 100
	lower, upper, ok = simulationPairedInterval(300, 300, pairs)
	if !ok || lower >= 46 || upper <= 54 {
		t.Fatalf("same-winner pairs must retain their increased variance: %v %v %v", lower, upper, ok)
	}
	if _, _, ok := simulationPairedInterval(0, 0, pairs); ok {
		t.Fatal("draws alone cannot estimate decided win rate")
	}
}
func (*mirrorCheckingStrategy) Attackers(mage.Player, *mage.Game) []uuid.UUID           { return nil }
func (*mirrorCheckingStrategy) Blockers(mage.Player, *mage.Game) []mage.BlockAssignment { return nil }

func TestSimulation_UsesMirrorDecks(t *testing.T) {
	t.Setenv("SIM_GAMES", "10")
	factory := func(ai.WeightedPersonality) ai.AIStrategy { return &mirrorCheckingStrategy{t: t} }
	run1kGames(t, "Mirror scheduling check", factory, factory)
}
