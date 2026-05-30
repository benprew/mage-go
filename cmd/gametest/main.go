// Command gametest runs two AI players against each other, printing the game
// state at the start of each turn and every decision each AI makes along with
// the choices it had available.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/internal/scenario"
	"github.com/benprew/mage-go/internal/tui"
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai/heuristic"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai/search"

	_ "github.com/benprew/mage-go/cards" // register all card sets
)

func main() {
	maxTurns := flag.Int("turns", 50, "maximum number of turns")
	deckA := flag.Int("deck-a", -1, "deck index for player A (0-based, -1 = random)")
	deckB := flag.Int("deck-b", -1, "deck index for player B (0-based, -1 = random)")
	rogueDecks := flag.Bool("rogue-decks", false, "select decks from Rogue .dck files instead of built-in archetypes")
	rogueDir := flag.String("rogue-dir", "rogue_dck", "path to Rogue .dck deck directory")
	minCards := flag.Int("min-cards", 25, "minimum playable cards for a Rogue deck to be eligible")
	games := flag.Int("games", 1, "number of games to play in sequence (useful for profiling)")
	seed := flag.Int64("seed", 0, "RNG seed for deck selection (0 = nondeterministic)")
	persA := flag.String("ai-a", "auto", "AI personality for player A (auto, aggro, control, midrange, tempo, burn)")
	persB := flag.String("ai-b", "auto", "AI personality for player B (auto, aggro, control, midrange, tempo, burn)")
	modeA := flag.String("mode-a", "heuristic", "AI mode for player A (heuristic, legacy, search, adaptive)")
	modeB := flag.String("mode-b", "heuristic", "AI mode for player B (heuristic, legacy, search, adaptive)")
	cpuProfile := flag.String("cpuprofile", "", "write cpu profile to file")
	memProfile := flag.String("memprofile", "", "write memory profile to file")
	timeout := flag.Duration("timeout", 0, "wall clock timeout for the whole run (0 disables)")
	loopTiming := flag.Bool("loop-timing", false, "write aggregate main game-loop timing to stderr")
	verbose := flag.Bool("verbose", false, "write per-search priority telemetry to stderr")
	quiet := flag.Bool("quiet", false, "suppress per-action game log output")
	flag.Parse()

	search.DebugStats = *verbose

	profiles := &profileRun{memPath: *memProfile}
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not create cpu profile: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "could not start cpu profile: %v\n", err)
			f.Close()
			os.Exit(1)
		}
		profiles.cpuFile = f
		profiles.cpuActive = true
	}
	defer profiles.finish()

	if *timeout > 0 {
		startWallclockTimeout(*timeout, profiles)
	}

	if *quiet {
		devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not open /dev/null: %v\n", err)
			os.Exit(1)
		}
		os.Stdout = devnull
	}

	deckPool, err := loadDeckPool(*rogueDecks, *rogueDir, *minCards)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not load decks: %v\n", err)
		os.Exit(1)
	}
	if *deckA >= len(deckPool) || *deckB >= len(deckPool) {
		fmt.Fprintf(os.Stderr, "invalid deck index (max %d)\n", len(deckPool)-1)
		os.Exit(1)
	}

	var rng *rand.Rand
	if *seed != 0 {
		rng = rand.New(rand.NewSource(*seed))
	} else {
		rng = rand.New(rand.NewSource(rand.Int63()))
	}

	totalLoop := time.Duration(0)

	for gameNum := 1; gameNum <= *games; gameNum++ {
		dA := *deckA
		if dA < 0 {
			dA = rng.Intn(len(deckPool))
		}
		dB := *deckB
		if dB < 0 {
			dB = rng.Intn(len(deckPool))
		}
		selectedA := deckPool[dA]
		selectedB := deckPool[dB]
		wpA := resolvePersonality(*persA, selectedA.Entries)
		wpB := resolvePersonality(*persB, selectedB.Entries)

		playerA := createAI("Alice", wpA, *modeA)
		playerB := createAI("Bob", wpB, *modeB)

		cardsA := tui.BuildDeck(selectedA.Entries, playerA.PlayerID())
		cardsB := tui.BuildDeck(selectedB.Entries, playerB.PlayerID())
		for _, c := range cardsA {
			playerA.AddToLibrary(c)
		}
		for _, c := range cardsB {
			playerB.AddToLibrary(c)
		}

		g := mage.NewGame(playerA, playerB)
		tui.DrawOpeningHand(playerA)
		tui.DrawOpeningHand(playerB)
		ai.MulliganAI(playerA)
		ai.MulliganAI(playerB)

		fmt.Printf("=== Game %d Start ===\n", gameNum)
		fmt.Printf("Alice (%s/%s) deck: %s (%d cards)\n", personalityLabel(*persA, wpA), *modeA, selectedA.Name, len(playerA.Library())+len(playerA.Hand()))
		printSkippedCards("Alice", selectedA.Skipped)
		fmt.Printf("Bob   (%s/%s) deck: %s (%d cards)\n", personalityLabel(*persB, wpB), *modeB, selectedB.Name, len(playerB.Library())+len(playerB.Hand()))
		printSkippedCards("Bob", selectedB.Skipped)
		fmt.Printf("Alice hand (%d): %s\n", len(playerA.Hand()), handStr(playerA.Hand()))
		fmt.Printf("Bob   hand (%d): %s\n\n", len(playerB.Hand()), handStr(playerB.Hand()))

		totalLoop += runGame(g, *maxTurns, *quiet)
	}
	if *loopTiming {
		fmt.Fprintf(
			os.Stderr,
			"loop_elapsed_s=%.6f games=%d loop_games_per_s=%.3f\n",
			totalLoop.Seconds(),
			*games,
			float64(*games)/max(totalLoop.Seconds(), 1e-9),
		)
	}
}

type profileRun struct {
	cpuFile   *os.File
	cpuActive bool
	memPath   string
	once      sync.Once
}

func (p *profileRun) finish() {
	p.once.Do(func() {
		if p.cpuActive {
			pprof.StopCPUProfile()
			p.cpuActive = false
		}
		if p.cpuFile != nil {
			if err := p.cpuFile.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "could not close cpu profile: %v\n", err)
			}
			p.cpuFile = nil
		}
		if p.memPath == "" {
			return
		}
		f, err := os.Create(p.memPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not create mem profile: %v\n", err)
			return
		}
		defer f.Close()
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "could not write mem profile: %v\n", err)
		}
	})
}

func startWallclockTimeout(d time.Duration, profiles *profileRun) {
	go func() {
		<-time.After(d)
		fmt.Fprintf(os.Stderr, "gametest wall clock timeout after %s; writing profiles and exiting\n", d)
		profiles.finish()
		os.Exit(124)
	}()
}

type selectableDeck struct {
	Name    string
	Entries []tui.DeckEntry
	Skipped []string
}

func loadDeckPool(useRogue bool, rogueDir string, minCards int) ([]selectableDeck, error) {
	if !useRogue {
		decks := make([]selectableDeck, 0, len(tui.Archetypes))
		for _, archetype := range tui.Archetypes {
			decks = append(decks, selectableDeck{Name: archetype.Name, Entries: archetype.Entries})
		}
		return decks, nil
	}

	rogueDecks, err := scenario.LoadAllDCKDecks(rogueDir)
	if err != nil {
		return nil, err
	}
	decks := make([]selectableDeck, 0, len(rogueDecks))
	for _, deck := range rogueDecks {
		entries, skipped := filterRogueDeck(deck)
		total := countDeckCards(entries)
		if total < minCards {
			fmt.Fprintf(os.Stderr, "skipping %s: only %d playable cards (need %d)\n", deck.Name, total, minCards)
			continue
		}
		decks = append(decks, selectableDeck{
			Name:    fmt.Sprintf("%s (%s)", deck.Name, deck.SourceFile),
			Entries: entries,
			Skipped: skipped,
		})
	}
	if len(decks) == 0 {
		return nil, fmt.Errorf("no eligible Rogue decks found in %s", rogueDir)
	}
	fmt.Fprintf(os.Stderr, "loaded %d eligible Rogue decks from %s\n", len(decks), rogueDir)
	return decks, nil
}

func filterRogueDeck(deck *scenario.RogueDeck) (entries []tui.DeckEntry, skipped []string) {
	primaryBasic := primaryBasicLand(deck.MainCards)
	backfill := 0
	for _, entry := range deck.MainCards {
		if cardAvailable(entry.Name) {
			entries = append(entries, tui.DeckEntry{Name: entry.Name, Count: entry.Count})
		} else {
			skipped = append(skipped, entry.Name)
			backfill += entry.Count
		}
	}
	if backfill > 0 {
		for i, entry := range entries {
			if entry.Name == primaryBasic {
				entries[i].Count += backfill
				return entries, skipped
			}
		}
		entries = append(entries, tui.DeckEntry{Name: primaryBasic, Count: backfill})
	}
	return entries, skipped
}

func primaryBasicLand(entries []scenario.DeckEntry) string {
	basics := map[string]int{}
	for _, entry := range entries {
		switch entry.Name {
		case "Plains", "Island", "Swamp", "Mountain", "Forest":
			basics[entry.Name] += entry.Count
		}
	}
	primary := "Plains"
	maxCount := 0
	for name, count := range basics {
		if count > maxCount {
			primary = name
			maxCount = count
		}
	}
	return primary
}

func cardAvailable(name string) bool {
	_, err := mage.CreateCard(name)
	return err == nil
}

func countDeckCards(entries []tui.DeckEntry) int {
	total := 0
	for _, entry := range entries {
		total += entry.Count
	}
	return total
}

func printSkippedCards(playerName string, skipped []string) {
	if len(skipped) == 0 {
		return
	}
	fmt.Printf("%s replacements: %d unsupported card names replaced with basics: %s\n",
		playerName, len(skipped), strings.Join(skipped, ", "))
}

func runGame(g *mage.Game, maxTurns int, quiet bool) time.Duration {

	// Track last printed turn so we print the header once.
	lastTurn := -1

	// Install priority handler: log choices and the AI's decision.
	g.SetOnPriority(func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		p := g.PlayerAt(playerIdx)
		aiPlayer := p.(interactive.AutoPlayer)

		// Print turn header on first priority of a new turn.
		if !quiet && g.CurrentTurn() != lastTurn {
			lastTurn = g.CurrentTurn()
			printTurnHeader(g)
		}

		// Only the active player gets main-phase privileges (land plays, sorceries).
		isActive := playerIdx == g.ActivePlayerIndex()
		effectiveMainPhase := mainPhase && isActive

		// Get the AI's decision.
		action := aiPlayer.GetPriorityAction(g, g.GetLandsPlayedThisTurn(), effectiveMainPhase)

		if quiet {
			return convertAction(action)
		}

		// Get available actions for display.
		options := interactive.GetAvailableActions(g, p.PlayerID())

		// Only print non-pass actions (or pass when there were real choices).
		hasRealChoice := len(options) > 1 || (len(options) == 1 && options[0].Type != interactive.ActionPass)
		if action.Type != interactive.ActionPass || hasRealChoice {
			if action.Type != interactive.ActionPass {
				phase := "priority"
				if mainPhase {
					phase = "main phase"
				}
				fmt.Printf("  [%s · %s · %s] %s decides:\n", g.GetStep(), phase, p.Name(), p.Name())
				printOptions(options)
				fmt.Printf("    → %s\n", describeAction(g, action))
			}
		}

		return convertAction(action)
	})

	if !quiet {
		// Log actions after execution.
		g.SetAfterPriorityAction(func(g *mage.Game, playerIdx int, action mage.PriorityAction) {
			p := g.PlayerAt(playerIdx)
			switch action.Type {
			case mage.PriorityPlayLand:
				perm := g.FindPermanent(action.CardID)
				name := "a land"
				if perm != nil {
					name = perm.Name()
				}
				fmt.Printf("    ✓ %s plays %s\n", p.Name(), name)
			case mage.PriorityCastSpell:
				obj := g.StackPeek()
				name := "a spell"
				if obj != nil && obj.Card != nil {
					name = obj.Card.Name()
				}
				fmt.Printf("    ✓ %s casts %s%s\n", p.Name(), name, targetSuffix(g, action.Targets))
			case mage.PriorityActivateAbility:
				perm := g.FindPermanent(action.PermanentID)
				name := "permanent"
				if perm != nil {
					name = perm.Name()
				}
				fmt.Printf("    ✓ Activated %s%s\n", name, targetSuffix(g, action.Targets))
			}
		})

		// Log stack resolution.
		g.SetBeforeStackResolve(func(g *mage.Game) {
			top := g.StackPeek()
			if top == nil {
				return
			}
			name := "ability"
			if top.Card != nil {
				name = top.Card.Name()
			}
			fmt.Printf("  ⟳ Resolving %s%s\n", name, targetSuffix(g, top.Targets))
		})
	}

	// Main turn loop.
	loopStart := time.Now()
	for g.CurrentTurn() <= maxTurns {
		for _, step := range core.AllSteps() {
			// Combat preview.
			if !quiet && step == core.CombatDamage && len(g.CombatGroups()) > 0 {
				fmt.Printf("  ── Combat damage ──\n")
				printCombatPreview(g)
			}

			g.RunStepWithPriority(step)

			// Post-step info.
			if !quiet {
				switch step {
				case core.Draw:
					fmt.Printf("  %s draws a card\n", g.ActivePlayerObj().Name())
				case core.CombatDamage:
					if len(g.CombatGroups()) > 0 {
						for _, p := range g.AllPlayers() {
							fmt.Printf("  %s: %d life\n", p.Name(), p.Life())
						}
					}
				}
			}

			if g.IsGameOver() {
				if !quiet {
					fmt.Printf("\n=== Game Over ===\n")
					fmt.Printf("Winner: %s\n", g.Winner())
					printFinalState(g)
				}
				return time.Since(loopStart)
			}
		}

		if g.HasExtraTurns() {
			extraPlayerID, _ := g.PopExtraTurn()
			for i, p := range g.AllPlayers() {
				if p.PlayerID() == extraPlayerID {
					g.SetActivePlayerIndex(i)
					break
				}
			}
		} else {
			g.SetActivePlayerIndex((g.ActivePlayerIndex() + 1) % g.PlayerCount())
		}
		g.SetTurn(g.CurrentTurn() + 1)
	}
	if !quiet {
		fmt.Printf("\n=== Game ended after %d turns (no winner) ===\n", maxTurns)
		printFinalState(g)
	}
	return time.Since(loopStart)
}

func createAI(name string, wp ai.WeightedPersonality, mode string) *ai.AIPlayer {
	switch strings.ToLower(mode) {
	case "search", "minimax":
		return ai.NewAIPlayer(name, search.New(search.DefaultConfig(), wp))
	case "adaptive":
		return ai.NewAIPlayer(name, heuristic.NewAdaptive())
	case "legacy", "old":
		return ai.NewAIPlayer(name, heuristic.NewLegacy(wp))
	default:
		return ai.NewAIPlayer(name, heuristic.New(wp))
	}
}

func parsePersonality(s string) ai.WeightedPersonality {
	switch strings.ToLower(s) {
	case "aggro":
		return ai.AggroWeighted
	case "control":
		return ai.ControlWeighted
	case "midrange":
		return ai.MidrangeWeighted
	case "tempo":
		return ai.TempoWeighted
	case "burn":
		return ai.BurnWeighted
	default:
		fmt.Fprintf(os.Stderr, "unknown personality %q, using midrange\n", s)
		return ai.MidrangeWeighted
	}
}

func resolvePersonality(s string, entries []tui.DeckEntry) ai.WeightedPersonality {
	if strings.EqualFold(s, "auto") || strings.EqualFold(s, "deck") {
		return ai.InferPersonalityFromDeck(aiDeckEntries(entries))
	}
	return parsePersonality(s)
}

func personalityLabel(requested string, wp ai.WeightedPersonality) string {
	if strings.EqualFold(requested, "auto") || strings.EqualFold(requested, "deck") {
		return "auto:" + strings.ToLower(wp.Name)
	}
	return requested
}

func aiDeckEntries(entries []tui.DeckEntry) []ai.DeckCard {
	deck := make([]ai.DeckCard, 0, len(entries))
	for _, entry := range entries {
		deck = append(deck, ai.DeckCard{Name: entry.Name, Count: entry.Count})
	}
	return deck
}

func handStr(hand []mage.Card) string {
	names := make([]string, len(hand))
	for i, c := range hand {
		names[i] = c.Name()
	}
	return strings.Join(names, ", ")
}

func printTurnHeader(g *mage.Game) {
	active := g.ActivePlayerObj()
	fmt.Printf("\n════ Turn %d: %s ════\n", g.CurrentTurn(), active.Name())
	for _, p := range g.AllPlayers() {
		fmt.Printf("  %s: %d life, %d in library, hand: [%s]\n",
			p.Name(), p.Life(), len(p.Library()), handStr(p.Hand()))
	}
	// Battlefield
	for _, p := range g.AllPlayers() {
		perms := battlefieldFor(g, p.PlayerID())
		if len(perms) > 0 {
			fmt.Printf("  %s's battlefield: %s\n", p.Name(), permStr(g, perms))
		}
	}
}

func battlefieldFor(g *mage.Game, playerID uuid.UUID) []*mage.Permanent {
	var perms []*mage.Permanent
	for _, perm := range g.AllBattlefield() {
		if perm.Controller == playerID {
			perms = append(perms, perm)
		}
	}
	return perms
}

func permStr(g *mage.Game, perms []*mage.Permanent) string {
	parts := make([]string, len(perms))
	for i, p := range perms {
		s := p.Name()
		if p.HasType(core.TypeCreature) {
			s += fmt.Sprintf(" %d/%d", p.CurrentPower(g), p.CurrentToughness(g))
		}
		if p.Tapped {
			s += " (T)"
		}
		parts[i] = s
	}
	return strings.Join(parts, ", ")
}

func printOptions(options []interactive.ActionOption) {
	for i, opt := range options {
		fmt.Printf("    %d. %s\n", i+1, opt.Label)
	}
}

func describeAction(g *mage.Game, action interactive.PriorityAction) string {
	switch action.Type {
	case interactive.ActionPlayLand:
		return fmt.Sprintf("Play land: %s", action.CardName)
	case interactive.ActionCastSpell:
		s := fmt.Sprintf("Cast %s", action.CardName)
		if len(action.Targets) > 0 {
			s += targetSuffix(g, action.Targets)
		}
		return s
	case interactive.ActionActivateAbility:
		perm := g.FindPermanent(action.PermanentID)
		name := "permanent"
		if perm != nil {
			name = perm.Name()
		}
		s := fmt.Sprintf("Activate %s", name)
		if len(action.Targets) > 0 {
			s += targetSuffix(g, action.Targets)
		}
		return s
	case interactive.ActionPass:
		return "Pass"
	default:
		return action.Type.String()
	}
}

func targetSuffix(g *mage.Game, targets []uuid.UUID) string {
	if len(targets) == 0 {
		return ""
	}
	names := make([]string, len(targets))
	for i, id := range targets {
		if perm := g.FindPermanent(id); perm != nil {
			names[i] = perm.Name()
		} else if player := g.GetPlayer(id); player != nil {
			names[i] = player.Name()
		} else {
			names[i] = "unknown"
		}
	}
	return " targeting " + strings.Join(names, ", ")
}

func printCombatPreview(g *mage.Game) {
	for _, grp := range g.CombatGroups() {
		atk := g.FindPermanent(grp.AttackerID)
		if atk == nil {
			continue
		}
		defender := g.GetPlayer(grp.DefenderID)
		defName := "player"
		if defender != nil {
			defName = defender.Name()
		}
		if len(grp.BlockerIDs) == 0 {
			fmt.Printf("    %s (%d/%d) → %s unblocked\n",
				atk.Name(), atk.CurrentPower(g), atk.CurrentToughness(g), defName)
		} else {
			parts := make([]string, 0, len(grp.BlockerIDs))
			for _, bid := range grp.BlockerIDs {
				blk := g.FindPermanent(bid)
				if blk != nil {
					parts = append(parts, fmt.Sprintf("%s (%d/%d)", blk.Name(), blk.CurrentPower(g), blk.CurrentToughness(g)))
				}
			}
			fmt.Printf("    %s (%d/%d) blocked by %s\n",
				atk.Name(), atk.CurrentPower(g), atk.CurrentToughness(g), strings.Join(parts, ", "))
		}
	}
}

func printFinalState(g *mage.Game) {
	for _, p := range g.AllPlayers() {
		fmt.Printf("  %s: %d life, %d cards in hand, %d in graveyard\n",
			p.Name(), p.Life(), len(p.Hand()), len(p.Graveyard()))
		perms := battlefieldFor(g, p.PlayerID())
		if len(perms) > 0 {
			fmt.Printf("    Battlefield: %s\n", permStr(g, perms))
		}
	}
}

func convertAction(action interactive.PriorityAction) mage.PriorityAction {
	switch action.Type {
	case interactive.ActionPlayLand:
		return mage.PriorityAction{Type: mage.PriorityPlayLand, CardID: action.CardID}
	case interactive.ActionCastSpell:
		return mage.PriorityAction{
			Type:    mage.PriorityCastSpell,
			CardID:  action.CardID,
			Targets: action.Targets,
			XValue:  action.XValue,
		}
	case interactive.ActionActivateAbility:
		return mage.PriorityAction{
			Type:        mage.PriorityActivateAbility,
			PermanentID: action.PermanentID,
			AbilityIdx:  action.AbilityIndex,
			Targets:     action.Targets,
		}
	default:
		return mage.PriorityAction{Type: mage.PriorityPass}
	}
}
