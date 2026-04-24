package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"

	_ "git.sr.ht/~cdcarter/mage-go/cards" // register all sets
)

func main() {
	var (
		xmageDir       string
		maxTurns       int
		seed           int64
		verbose        bool
		games          int
		outFile        string
		roguesDir      string
	)

	flag.StringVar(&xmageDir, "xmage", "../xmage", "path to XMage repo")
	flag.IntVar(&maxTurns, "turns", 10, "max turns per game")
	flag.Int64Var(&seed, "seed", 0, "random seed (0 = time-based)")
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.IntVar(&games, "games", 1, "number of games to run")
	flag.StringVar(&outFile, "out", "", "write divergence logs to this directory")
	flag.StringVar(&roguesDir, "rogues", "", "directory of rogue deck .toml files")
	flag.Parse()

	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(seed))

	fmt.Printf("Cross-validation: mage-go vs XMage (XMage-driven)\n")
	fmt.Printf("  Seed: %d, Turns: %d, Games: %d\n", seed, maxTurns, games)

	// Step 1: Launch oracle and get card intersection
	goCards := mage.RegisteredCardNames()
	sort.Strings(goCards)
	fmt.Printf("  mage-go cards: %d\n", len(goCards))

	fmt.Println("Launching XMage oracle...")
	oracle, err := launchOracle(xmageDir, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to launch oracle: %v\n", err)
		os.Exit(1)
	}
	defer oracle.close()

	if err := oracle.send(map[string]any{"type": "card_check", "cards": goCards}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send card check: %v\n", err)
		os.Exit(1)
	}

	resp, err := oracle.recv(300 * time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Card check failed: %v\n", err)
		os.Exit(1)
	}
	available := resp.Cards
	fmt.Printf("  Card intersection: %d\n", len(available))

	if outFile != "" {
		os.MkdirAll(outFile, 0755)
	}

	var rogueDecks []rogueDeck
	if roguesDir != "" {
		var err error
		rogueDecks, err = loadRogueDecks(roguesDir, available)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load rogue decks: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  Rogue decks loaded: %d\n", len(rogueDecks))
	}

	// Step 2: Run games
	totalDecisions := 0
	gamesOK := 0
	gamesDiverged := 0
	gameErrors := 0

	for gameNum := 0; gameNum < games; gameNum++ {
		if gameNum%10 == 0 {
			fmt.Printf("  Game %d/%d...\n", gameNum+1, games)
		}

		deckA := pickDeck(rogueDecks, available, rng)
		deckB := pickDeck(rogueDecks, available, rng)

		decisions, divergences, err := runXMageDrivenGame(oracle, deckA, deckB, maxTurns, rng, verbose)
		if err != nil {
			gameErrors++
			if verbose {
				fmt.Fprintf(os.Stderr, "  Game %d error: %v\n", gameNum+1, err)
			}
			if outFile != "" {
				writeErrorLog(outFile, gameNum+1, seed, deckA, deckB, err)
			}
			continue
		}

		totalDecisions += decisions
		if len(divergences) > 0 {
			gamesDiverged++
			fmt.Printf("  Game %d: %d divergences (out of %d decisions)\n",
				gameNum+1, len(divergences), decisions)
			if verbose {
				for _, d := range divergences {
					fmt.Printf("    %s\n", d)
				}
			}
			if outFile != "" {
				writeGameLog(outFile, gameNum+1, seed, deckA, deckB, divergences, decisions)
			}
		} else {
			gamesOK++
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Games: %d, OK: %d, Diverged: %d, Errors: %d\n",
		games, gamesOK, gamesDiverged, gameErrors)
	fmt.Printf("Total decisions: %d\n", totalDecisions)

	if outFile != "" && (gamesDiverged > 0 || gameErrors > 0) {
		f, _ := os.Create(outFile + "/summary.log")
		if f != nil {
			fmt.Fprintf(f, "seed=%d turns=%d games=%d\n", seed, maxTurns, games)
			fmt.Fprintf(f, "OK=%d diverged=%d errors=%d decisions=%d\n",
				gamesOK, gamesDiverged, gameErrors, totalDecisions)
			f.Close()
		}
		fmt.Printf("Logs written to %s/\n", outFile)
	}

	if gamesDiverged > 0 {
		os.Exit(1)
	}
}

// runXMageDrivenGame runs one game with the Go driver choosing actions at each
// XMage decision point. Both engines execute the same action sequence so their
// states can be compared.
func runXMageDrivenGame(oracle *xmageOracle, deckA, deckB []string, maxTurns int, rng *rand.Rand, verbose bool) (int, []string, error) {
	setup := setupMsg{
		Type:     "setup",
		PlayerA:  playerDef{Name: "Alice", Library: deckA},
		PlayerB:  playerDef{Name: "Bob", Library: deckB},
		HandSize: 7,
		MaxTurns: maxTurns,
	}
	if err := oracle.send(setup); err != nil {
		return 0, nil, fmt.Errorf("send setup: %w", err)
	}

	decisions := 0
	var divergences []string
	var lastState *cvState

	for {
		msg, err := oracle.recv(30 * time.Second)
		if err != nil {
			return decisions, divergences, fmt.Errorf("recv: %w", err)
		}

		if msg.Type == "game_over" {
			break
		}

		if msg.Type == "error" {
			return decisions, divergences, fmt.Errorf("oracle: %s", msg.Message)
		}

		if msg.Type != "decision_point" {
			continue
		}

		decisions++

		if msg.State != nil {
			lastState = msg.State

			for i := 0; i < 2; i++ {
				p := msg.State.Players[i]
				if p.Life < -100 {
					divergences = append(divergences,
						fmt.Sprintf("T%d %s: %s life=%d (runaway negative life)",
							msg.State.Turn, msg.State.Step, p.Name, p.Life))
				}
			}
		}

		var action actionMsg
		switch msg.Kind {
		case "priority":
			action = choosePriorityAction(msg.Legal, msg.State, rng)
		case "attackers", "declare_attackers":
			action = chooseAttackers(msg.Legal, msg.State)
		case "blockers", "declare_blockers":
			action = chooseBlockers(msg.Legal, msg.State)
		case "choice":
			action = chooseChoice(msg)
		default:
			action = actionMsg{Type: "action", Kind: "pass"}
		}

		if verbose {
			step := ""
			if msg.State != nil {
				step = fmt.Sprintf("T%d %s", msg.State.Turn, msg.State.Step)
			}
			fmt.Printf("    %s [%s] -> %s", step, msg.Kind, action.Kind)
			if action.CardName != "" {
				fmt.Printf(" %s", action.CardName)
			}
			if len(action.Targets) > 0 {
				fmt.Printf(" targets=%v", action.Targets)
			}
			if action.XValue > 0 {
				fmt.Printf(" X=%d", action.XValue)
			}
			if len(action.Attackers) > 0 {
				fmt.Printf(" attackers=%v", action.Attackers)
			}
			if len(action.Blockers) > 0 {
				fmt.Printf(" blockers=%v", action.Blockers)
			}
			fmt.Println()
		}

		if err := oracle.send(action); err != nil {
			return decisions, divergences, fmt.Errorf("send action: %w", err)
		}
	}

	_ = lastState
	return decisions, divergences, nil
}

func writeErrorLog(dir string, gameNum int, seed int64, deckA, deckB []string, gameErr error) {
	fname := fmt.Sprintf("%s/error_%04d.log", dir, gameNum)
	f, err := os.Create(fname)
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "# Game %d (seed=%d)\n", gameNum, seed)
	fmt.Fprintf(f, "# Error: %v\n\n", gameErr)
	fmt.Fprintf(f, "## Deck A\n")
	for _, c := range deckA {
		fmt.Fprintf(f, "  %s\n", c)
	}
	fmt.Fprintf(f, "\n## Deck B\n")
	for _, c := range deckB {
		fmt.Fprintf(f, "  %s\n", c)
	}
}

func writeGameLog(dir string, gameNum int, seed int64, deckA, deckB []string, divergences []string, decisions int) {
	fname := fmt.Sprintf("%s/game_%04d.log", dir, gameNum)
	f, err := os.Create(fname)
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "# Game %d (seed=%d)\n", gameNum, seed)
	fmt.Fprintf(f, "# %d divergences, %d decisions\n\n", len(divergences), decisions)
	fmt.Fprintf(f, "## Deck A\n")
	for _, c := range deckA {
		fmt.Fprintf(f, "  %s\n", c)
	}
	fmt.Fprintf(f, "\n## Deck B\n")
	for _, c := range deckB {
		fmt.Fprintf(f, "  %s\n", c)
	}
	fmt.Fprintf(f, "\n## Divergences\n")
	for _, d := range divergences {
		fmt.Fprintf(f, "%s\n", d)
	}
}

// parsePreState extracts the pre_state from the raw oracle message JSON.
func parsePreState(raw json.RawMessage) *cvState {
	var wrapper struct {
		PreState *cvState `json:"pre_state"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil
	}
	return wrapper.PreState
}

// Placeholder for future: deduplicate divergence messages
func dedup(msgs []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, m := range msgs {
		key := m
		// Normalize turn/step prefix for dedup
		if idx := strings.Index(m, ":"); idx > 0 {
			key = m[idx+1:]
		}
		if !seen[key] {
			seen[key] = true
			result = append(result, m)
		}
	}
	return result
}
