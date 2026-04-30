package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"time"

	_ "git.sr.ht/~cdcarter/mage-go/cards" // register all sets
	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
)

func main() {
	var (
		xmageDir    string
		maxTurns    int
		seed        int64
		verbose     bool
		debug       bool
		games       int
		outFile     string
		roguesDir   string
		gameTimeout int
	)

	flag.StringVar(&xmageDir, "xmage", "../xmage", "path to XMage repo")
	flag.IntVar(&maxTurns, "turns", 10, "max turns per game")
	flag.Int64Var(&seed, "seed", 0, "random seed (0 = time-based)")
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.BoolVar(&debug, "debug", false, "trace IPC and synchronization (helps diagnose deadlocks)")
	flag.IntVar(&games, "games", 1, "number of games to run")
	flag.StringVar(&outFile, "out", "", "write divergence logs to this directory")
	flag.StringVar(&roguesDir, "rogues", "", "directory of rogue deck .toml files")
	flag.IntVar(&gameTimeout, "game-timeout", 60, "per-game wall-clock timeout in seconds (0 disables)")
	flag.Parse()

	if debug {
		_ = os.Setenv("CROSSVAL_DEBUG", "1")
	}

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
	totalWarnings := 0
	gamesOK := 0
	gamesDiverged := 0
	gamesWithWarnings := 0
	gameErrors := 0
	gamesHung := 0

	for gameNum := 0; gameNum < games; gameNum++ {
		if gameNum%10 == 0 {
			fmt.Printf("  Game %d/%d...\n", gameNum+1, games)
		}

		deckA := pickDeck(rogueDecks, available, rng)
		deckB := pickDeck(rogueDecks, available, rng)

		// Per-game wall-clock guard. XMage's AI can get stuck in an infinite
		// decision loop on certain board states; without a hard deadline a
		// single hung game stalls the whole sweep. On timeout we force-kill
		// the JVM AND close abortCh — the latter is needed because the main
		// loop frequently blocks sending on the mirror's stepCh/msgCh, which
		// killing the JVM alone won't unblock (it only closes the recv pipe).
		abortCh := make(chan struct{})
		var timer *time.Timer
		if gameTimeout > 0 {
			timer = time.AfterFunc(time.Duration(gameTimeout)*time.Second, func() {
				fmt.Fprintf(os.Stderr, "  Game %d hung (>%ds), killing JVM\n", gameNum+1, gameTimeout)
				close(abortCh)
				if oracle.cmd != nil && oracle.cmd.Process != nil {
					oracle.cmd.Process.Kill()
				}
			})
		}
		decisions, divergences, warnings, err := runXMageDrivenGame(oracle, deckA, deckB, maxTurns, verbose, debug, abortCh)
		hung := timer != nil && !timer.Stop()
		if err != nil {
			gameErrors++
			if hung {
				gamesHung++
			}
			fmt.Fprintf(os.Stderr, "  Game %d error: %v\n", gameNum+1, err)
			for _, line := range oracle.lastErr.lines() {
				fmt.Fprintf(os.Stderr, "  [xmage] %s\n", line)
			}
			if outFile != "" {
				writeErrorLog(outFile, gameNum+1, seed, deckA, deckB, err)
			}
			// Oracle process may have crashed — restart it.
			oracle.close()
			oracle, err = launchOracle(xmageDir, verbose)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to relaunch oracle: %v\n", err)
				os.Exit(1)
			}
			if err := oracle.send(map[string]any{"type": "card_check", "cards": goCards}); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to resend card check: %v\n", err)
				os.Exit(1)
			}
			if _, err := oracle.recv(300 * time.Second); err != nil {
				fmt.Fprintf(os.Stderr, "Card check failed after relaunch: %v\n", err)
				os.Exit(1)
			}
			continue
		}

		totalDecisions += decisions
		totalWarnings += len(warnings)
		if len(warnings) > 0 {
			gamesWithWarnings++
		}
		if len(divergences) > 0 {
			gamesDiverged++
			fmt.Printf("  Game %d: %d divergences, %d warnings (out of %d decisions)\n",
				gameNum+1, len(divergences), len(warnings), decisions)
			if verbose {
				for _, d := range divergences {
					fmt.Printf("    %s\n", d)
				}
				for _, w := range warnings {
					fmt.Printf("    %s\n", w)
				}
			}
			if outFile != "" {
				writeGameLog(outFile, gameNum+1, seed, deckA, deckB, divergences, warnings, decisions)
			}
		} else {
			gamesOK++
			if len(warnings) > 0 && verbose {
				fmt.Printf("  Game %d: OK with %d warnings\n", gameNum+1, len(warnings))
				for _, w := range warnings {
					fmt.Printf("    %s\n", w)
				}
			}
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Games: %d, OK: %d, Diverged: %d, Errors: %d (Hung: %d)\n",
		games, gamesOK, gamesDiverged, gameErrors, gamesHung)
	fmt.Printf("Warnings: %d total across %d games\n", totalWarnings, gamesWithWarnings)
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

// runXMageDrivenGame runs one game where XMage drives all decisions.
// The Go engine mirrors each action and compares state after non-pass actions.
// Returns (decisions, divergences, warnings, err) where divergences are real
// state mismatches (engines disagree on a stable state) and warnings are
// transient mid-stack diffs that typically reflect CR-valid trigger-ordering
// choices rather than engine bugs.
func runXMageDrivenGame(oracle *xmageOracle, deckA, deckB []string, maxTurns int, verbose, debug bool, abortCh <-chan struct{}) (int, []string, []string, error) {
	setup := setupMsg{
		Type:     "setup",
		PlayerA:  playerDef{Name: "Alice", Library: deckA},
		PlayerB:  playerDef{Name: "Bob", Library: deckB},
		HandSize: 7,
		MaxTurns: maxTurns,
	}
	if err := oracle.send(setup); err != nil {
		return 0, nil, nil, fmt.Errorf("send setup: %w", err)
	}

	mg, err := newMirrorGame(deckA, deckB)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("create mirror game: %w", err)
	}
	mg.debug = debug
	mg.start()
	defer mg.stop()
	_ = maxTurns // XMage now drives step transitions; max_turns is enforced server-side via stopOnTurn.

	decisions := 0
	var divergences []string
	var warnings []string

	ack := ackMsg{Type: "ack"}

	for {
		dbg(debug, "main: waiting for oracle message...")
		msg, err := oracle.recv(120 * time.Second)
		if err != nil {
			dbg(debug, "main: recv error: %v (go-state=%s)", err, mg.snapshotState())
			return decisions, divergences, warnings, fmt.Errorf("recv: %w", err)
		}
		dbg(debug, "main: recv type=%s turn=%d step=%s player=%d",
			msg.Type, msg.Turn, msg.Step, msg.PlayerIdx)

		switch msg.Type {
		case "game_over":
			return decisions, divergences, warnings, nil

		case "error":
			return decisions, divergences, warnings, fmt.Errorf("oracle: %s", msg.Message)

		case "step_begin":
			step, ok := parseStep(msg.Step)
			if !ok {
				return decisions, divergences, warnings, fmt.Errorf("step_begin: unknown step %q", msg.Step)
			}
			dbg(debug, "main: -> stepCh (T%d %s active=%d)", msg.Turn, msg.Step, msg.ActivePlayerIdx)
			select {
			case mg.stepCh <- stepInfo{
				turn:            msg.Turn,
				step:            step,
				activePlayerIdx: msg.ActivePlayerIdx,
			}:
			case <-mg.doneCh:
				return decisions, divergences, warnings, fmt.Errorf(
					"go game exited before step_begin (xmage T%d %s, gameErr=%w)",
					msg.Turn, msg.Step, mg.gameErr)
			case <-abortCh:
				return decisions, divergences, warnings, fmt.Errorf(
					"aborted (xmage T%d %s) — likely game timeout", msg.Turn, msg.Step)
			}
			if err := oracle.send(ack); err != nil {
				return decisions, divergences, warnings, fmt.Errorf("send step_begin ack: %w", err)
			}

		case "action_taken":
			decisions++
			if msg.Action == nil {
				if err := oracle.send(ack); err != nil {
					return decisions, divergences, warnings, fmt.Errorf("send ack: %w", err)
				}
				continue
			}

			if verbose {
				fmt.Printf("    T%d %s p%d: %s",
					msg.Turn, msg.Step, msg.PlayerIdx, msg.Action.Kind)
				if msg.Action.CardName != "" {
					fmt.Printf(" %s", msg.Action.CardName)
				}
				if len(msg.Action.Targets) > 0 {
					fmt.Printf(" targets=%v", msg.Action.Targets)
				}
				if msg.Action.X > 0 {
					fmt.Printf(" x=%d", msg.Action.X)
				}
				fmt.Println()
			}

			dbg(debug, "main: -> msgCh (T%d %s p%d %s)", msg.Turn, msg.Step, msg.PlayerIdx, msg.Action.Kind)
			select {
			case mg.msgCh <- mirrorMsg{
				playerIdx: msg.PlayerIdx,
				action:    *msg.Action,
				step:      msg.Step,
			}:
			case <-mg.doneCh:
				return decisions, divergences, warnings, fmt.Errorf(
					"go game goroutine exited before xmage finished (xmage at T%d %s p%d, go-state=%s, gameErr=%w)",
					msg.Turn, msg.Step, msg.PlayerIdx, mg.snapshotState(), mg.gameErr)
			case <-abortCh:
				return decisions, divergences, warnings, fmt.Errorf(
					"aborted (xmage at T%d %s p%d) — likely game timeout", msg.Turn, msg.Step, msg.PlayerIdx)
			}
			dbg(debug, "main: msgCh sent, waiting on resultCh")

			var result mirrorResult
			select {
			case result = <-mg.resultCh:
			case <-mg.doneCh:
				return decisions, divergences, warnings, fmt.Errorf(
					"go game goroutine exited while waiting for resultCh (xmage at T%d %s, go-state=%s, gameErr=%w)",
					msg.Turn, msg.Step, mg.snapshotState(), mg.gameErr)
			case <-abortCh:
				return decisions, divergences, warnings, fmt.Errorf(
					"aborted (xmage at T%d %s, awaiting resultCh) — likely game timeout", msg.Turn, msg.Step)
			}
			dbg(debug, "main: <- resultCh (err=%v go-step=%s)", result.err, mg.snapshotState())
			if result.err != nil {
				return decisions, divergences, warnings, fmt.Errorf("mirror error: %w", result.err)
			}

			if result.state != nil && msg.State != nil {
				mm := compareStates(result.state, msg.State)
				for _, m := range mm {
					if m.Warning {
						warnings = append(warnings, m.String())
					} else {
						divergences = append(divergences, m.String())
					}
				}
			}

			if err := oracle.send(ack); err != nil {
				return decisions, divergences, warnings, fmt.Errorf("send ack: %w", err)
			}

		case "attackers_declared":
			decisions++
			if verbose {
				fmt.Printf("    T%d attackers: %v\n", msg.Turn, msg.Attackers)
			}
			dbg(debug, "main: -> attackCh[p%d] (n=%d) go-state=%s",
				msg.PlayerIdx, len(msg.Attackers), mg.snapshotState())
			select {
			case mg.players[msg.PlayerIdx].attackCh <- msg.Attackers:
			case <-mg.doneCh:
				return decisions, divergences, warnings, fmt.Errorf(
					"go game exited before attackers fed (xmage at T%d, go-state=%s, gameErr=%w)",
					msg.Turn, mg.snapshotState(), mg.gameErr)
			case <-abortCh:
				return decisions, divergences, warnings, fmt.Errorf(
					"aborted (xmage at T%d, attackers) — likely game timeout", msg.Turn)
			}
			dbg(debug, "main: attackCh sent")

			if err := oracle.send(ack); err != nil {
				return decisions, divergences, warnings, fmt.Errorf("send ack: %w", err)
			}

		case "blockers_declared":
			decisions++
			if verbose {
				fmt.Printf("    T%d blockers: %v\n", msg.Turn, msg.Blockers)
			}
			dbg(debug, "main: -> blockCh[p%d] (n=%d) go-state=%s",
				msg.PlayerIdx, len(msg.Blockers), mg.snapshotState())
			select {
			case mg.players[msg.PlayerIdx].blockCh <- msg.Blockers:
			case <-mg.doneCh:
				return decisions, divergences, warnings, fmt.Errorf(
					"go game exited before blockers fed (xmage at T%d, go-state=%s, gameErr=%w)",
					msg.Turn, mg.snapshotState(), mg.gameErr)
			case <-abortCh:
				return decisions, divergences, warnings, fmt.Errorf(
					"aborted (xmage at T%d, blockers) — likely game timeout", msg.Turn)
			}
			dbg(debug, "main: blockCh sent")

			if err := oracle.send(ack); err != nil {
				return decisions, divergences, warnings, fmt.Errorf("send ack: %w", err)
			}

		default:
			// Ignore unknown message types
		}
	}
}

func dbg(enabled bool, format string, args ...any) {
	if enabled {
		fmt.Fprintf(os.Stderr, "[crossval] "+format+"\n", args...)
	}
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

func writeGameLog(dir string, gameNum int, seed int64, deckA, deckB []string, divergences, warnings []string, decisions int) {
	fname := fmt.Sprintf("%s/game_%04d.log", dir, gameNum)
	f, err := os.Create(fname)
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "# Game %d (seed=%d)\n", gameNum, seed)
	fmt.Fprintf(f, "# %d divergences, %d warnings, %d decisions\n\n", len(divergences), len(warnings), decisions)
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
	if len(warnings) > 0 {
		fmt.Fprintf(f, "\n## Warnings (transient mid-stack diffs)\n")
		for _, w := range warnings {
			fmt.Fprintf(f, "%s\n", w)
		}
	}
}
