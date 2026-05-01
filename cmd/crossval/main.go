package main

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "git.sr.ht/~cdcarter/mage-go/cards" // register all sets
	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
)

// errStoppedAtDivergence is returned by runXMageDrivenGame when it bails out
// after detecting the first divergence. Treated as a divergence (not an error)
// in the summary, but still triggers an oracle restart since we've left the
// game half-finished on xmage's side.
var errStoppedAtDivergence = errors.New("stopped at first divergence")

// gameRun aggregates everything one game run produces. Returned by
// runXMageDrivenGame so callers don't juggle a 6-tuple.
type gameRun struct {
	numDecisions int
	divergences  []string // stable-state mismatches
	warnings     []string // mid-stack diffs (CR-valid trigger ordering)
	decisions    []string // formatted action history (one line per decision)
	dump         string   // first-divergence dump; empty if game didn't diverge
}

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
		gameOnly    int
		dumpDir     string
		replayDir   string
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
	flag.IntVar(&gameOnly, "game-only", 0, "if >0, run only game N (1-indexed); earlier games still consume RNG so deck picks match a full -games run")
	flag.StringVar(&dumpDir, "dump", "", "if set, dump xmage-only games as JSONL transcripts into this directory and skip mage-go mirroring/divergence checks entirely")
	flag.StringVar(&replayDir, "replay-dir", "", "if set, replay recorded transcripts from this directory through mage-go (no JVM, no IPC); -dump-format files are valid transcripts")
	flag.Parse()

	if replayDir != "" {
		runReplayMode(replayDir, outFile, gameOnly, verbose, debug)
		return
	}

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
	if dumpDir != "" {
		os.MkdirAll(dumpDir, 0755)
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
		// pickDeck must run on every iteration even when --game-only skips
		// the actual game so the RNG state matches a full -games run.
		deckA := pickDeck(rogueDecks, available, rng)
		deckB := pickDeck(rogueDecks, available, rng)
		if gameOnly > 0 && gameNum+1 != gameOnly {
			continue
		}
		if gameNum%10 == 0 || gameOnly > 0 {
			fmt.Printf("  Game %d/%d...\n", gameNum+1, games)
		}

		if dumpDir != "" {
			// Dump mode: take mage-go fully out of the per-message loop.
			// XMage drives the whole game with its AI; we just persist its
			// JSON wire output verbatim and ack so xmage keeps moving.
			var timer *time.Timer
			if gameTimeout > 0 {
				timer = time.AfterFunc(time.Duration(gameTimeout)*time.Second, func() {
					fmt.Fprintf(os.Stderr, "  Game %d hung (>%ds), killing JVM\n", gameNum+1, gameTimeout)
					if oracle.cmd != nil && oracle.cmd.Process != nil {
						oracle.cmd.Process.Kill()
					}
				})
			}
			dumpPath := filepath.Join(dumpDir, fmt.Sprintf("game_%04d.jsonl", gameNum+1))
			err := runXMageDumpGame(oracle, deckA, deckB, maxTurns, dumpPath, seed, gameNum+1)
			hung := timer != nil && !timer.Stop()
			if err != nil {
				gameErrors++
				if hung {
					gamesHung++
				}
				fmt.Fprintf(os.Stderr, "  Game %d dump error: %v\n", gameNum+1, err)
				for _, line := range oracle.lastErr.lines() {
					fmt.Fprintf(os.Stderr, "  [xmage] %s\n", line)
				}
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
			} else {
				gamesOK++
			}
			if gameOnly > 0 {
				break
			}
			continue
		}

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
		run, err := runXMageDrivenGame(oracle, deckA, deckB, maxTurns, verbose, debug, abortCh)
		hung := timer != nil && !timer.Stop()

		stoppedAtDivergence := errors.Is(err, errStoppedAtDivergence)
		if err != nil && !stoppedAtDivergence {
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
				writeDecisionLog(outFile, gameNum+1, run.decisions)
			}
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

		totalDecisions += run.numDecisions
		totalWarnings += len(run.warnings)
		if len(run.warnings) > 0 {
			gamesWithWarnings++
		}
		if len(run.divergences) > 0 {
			gamesDiverged++
			fmt.Printf("  Game %d: %d divergences, %d warnings (out of %d decisions, stopped early=%v)\n",
				gameNum+1, len(run.divergences), len(run.warnings), run.numDecisions, stoppedAtDivergence)
			if verbose {
				for _, d := range run.divergences {
					fmt.Printf("    %s\n", d)
				}
				for _, w := range run.warnings {
					fmt.Printf("    %s\n", w)
				}
			}
			if outFile != "" {
				writeGameLog(outFile, gameNum+1, seed, deckA, deckB, run.divergences, run.warnings, run.numDecisions)
				writeDecisionLog(outFile, gameNum+1, run.decisions)
				if run.dump != "" {
					writeFirstDivergenceLog(outFile, gameNum+1, run.dump)
				}
			}
		} else {
			gamesOK++
			if len(run.warnings) > 0 && verbose {
				fmt.Printf("  Game %d: OK with %d warnings\n", gameNum+1, len(run.warnings))
				for _, w := range run.warnings {
					fmt.Printf("    %s\n", w)
				}
			}
		}

		// On stop-at-divergence we left xmage mid-game; restart the oracle so
		// the next game starts clean. Same recovery as the error path above.
		if stoppedAtDivergence {
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
		}

		if gameOnly > 0 {
			break
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
// Returns a gameRun describing what happened and an error if the game aborted.
// On the first stable-state divergence the game is stopped early (with err =
// errStoppedAtDivergence): once the engines disagree on a permanent's zone or
// life total, every subsequent comparison is contaminated by that drift, so
// further data is noise.
//
// The src parameter abstracts where events come from: a live xmageOracle (we
// drive a JVM and ack each event) or a replaySource (we read a recorded
// transcript off disk). The function body only ever calls src.send / src.recv,
// so the two modes share all comparison/dump/decision-log logic.
func runXMageDrivenGame(src eventSource, deckA, deckB []string, maxTurns int, verbose, debug bool, abortCh <-chan struct{}) (gameRun, error) {
	run := gameRun{}

	setup := setupMsg{
		Type:     "setup",
		PlayerA:  playerDef{Name: "Alice", Library: deckA},
		PlayerB:  playerDef{Name: "Bob", Library: deckB},
		HandSize: 7,
		MaxTurns: maxTurns,
	}
	if err := src.send(setup); err != nil {
		return run, fmt.Errorf("send setup: %w", err)
	}

	mg, err := newMirrorGame(deckA, deckB)
	if err != nil {
		return run, fmt.Errorf("create mirror game: %w", err)
	}
	mg.debug = debug
	mg.start()
	defer mg.stop()
	_ = maxTurns // XMage now drives step transitions; max_turns is enforced server-side via stopOnTurn.

	ack := ackMsg{Type: "ack"}

	for {
		dbg(debug, "main: waiting for oracle message...")
		msg, err := src.recv(120 * time.Second)
		if err != nil {
			dbg(debug, "main: recv error: %v (go-state=%s)", err, mg.snapshotState())
			return run, fmt.Errorf("recv: %w", err)
		}
		dbg(debug, "main: recv type=%s turn=%d step=%s player=%d",
			msg.Type, msg.Turn, msg.Step, msg.PlayerIdx)

		switch msg.Type {
		case "game_over":
			return run, nil

		case "error":
			return run, fmt.Errorf("oracle: %s", msg.Message)

		case "step_begin":
			step, ok := parseStep(msg.Step)
			if !ok {
				return run, fmt.Errorf("step_begin: unknown step %q", msg.Step)
			}
			dbg(debug, "main: -> stepCh (T%d %s active=%d)", msg.Turn, msg.Step, msg.ActivePlayerIdx)
			select {
			case mg.stepCh <- stepInfo{
				turn:            msg.Turn,
				step:            step,
				activePlayerIdx: msg.ActivePlayerIdx,
			}:
			case <-mg.doneCh:
				return run, fmt.Errorf(
					"go game exited before step_begin (xmage T%d %s, gameErr=%w)",
					msg.Turn, msg.Step, mg.gameErr)
			case <-abortCh:
				return run, fmt.Errorf(
					"aborted (xmage T%d %s) — likely game timeout", msg.Turn, msg.Step)
			}
			if err := src.send(ack); err != nil {
				return run, fmt.Errorf("send step_begin ack: %w", err)
			}

		case "action_taken":
			run.numDecisions++
			if msg.Action == nil {
				if err := src.send(ack); err != nil {
					return run, fmt.Errorf("send ack: %w", err)
				}
				continue
			}

			line := formatActionLine(msg.Turn, msg.Step, msg.PlayerIdx, msg.Action)
			run.decisions = append(run.decisions, line)
			if verbose {
				fmt.Printf("    %s\n", line)
			}

			dbg(debug, "main: -> msgCh (T%d %s p%d %s)", msg.Turn, msg.Step, msg.PlayerIdx, msg.Action.Kind)
			select {
			case mg.msgCh <- mirrorMsg{
				playerIdx: msg.PlayerIdx,
				action:    *msg.Action,
				step:      msg.Step,
			}:
			case <-mg.doneCh:
				return run, fmt.Errorf(
					"go game goroutine exited before xmage finished (xmage at T%d %s p%d, go-state=%s, gameErr=%w)",
					msg.Turn, msg.Step, msg.PlayerIdx, mg.snapshotState(), mg.gameErr)
			case <-abortCh:
				return run, fmt.Errorf(
					"aborted (xmage at T%d %s p%d) — likely game timeout", msg.Turn, msg.Step, msg.PlayerIdx)
			}
			dbg(debug, "main: msgCh sent, waiting on resultCh")

			var result mirrorResult
			select {
			case result = <-mg.resultCh:
			case <-mg.doneCh:
				return run, fmt.Errorf(
					"go game goroutine exited while waiting for resultCh (xmage at T%d %s, go-state=%s, gameErr=%w)",
					msg.Turn, msg.Step, mg.snapshotState(), mg.gameErr)
			case <-abortCh:
				return run, fmt.Errorf(
					"aborted (xmage at T%d %s, awaiting resultCh) — likely game timeout", msg.Turn, msg.Step)
			}
			dbg(debug, "main: <- resultCh (err=%v go-step=%s)", result.err, mg.snapshotState())
			if result.err != nil {
				return run, fmt.Errorf("mirror error: %w", result.err)
			}

			if result.state != nil && msg.State != nil {
				mm := compareStates(result.state, msg.State)
				priorDivergences := len(run.divergences)
				for _, m := range mm {
					if m.Warning {
						run.warnings = append(run.warnings, m.String())
					} else {
						run.divergences = append(run.divergences, m.String())
					}
				}
				// First stable-state divergence: dump both engines' view of
				// the world plus the action that triggered it, and bail. We
				// keep the divergences we've already seen at this comparison
				// (often several fields differ on the same action).
				if priorDivergences == 0 && len(run.divergences) > 0 {
					run.dump = buildFirstDivergenceDump(msg, line, result.state, msg.State, run.divergences, run.decisions)
					return run, errStoppedAtDivergence
				}
			}

			if err := src.send(ack); err != nil {
				return run, fmt.Errorf("send ack: %w", err)
			}

		case "attackers_declared":
			run.numDecisions++
			line := fmt.Sprintf("T%d attackers: %v", msg.Turn, msg.Attackers)
			run.decisions = append(run.decisions, line)
			if verbose {
				fmt.Printf("    %s\n", line)
			}
			dbg(debug, "main: -> attackCh[p%d] (n=%d) go-state=%s",
				msg.PlayerIdx, len(msg.Attackers), mg.snapshotState())
			select {
			case mg.players[msg.PlayerIdx].attackCh <- msg.Attackers:
			case <-mg.doneCh:
				return run, fmt.Errorf(
					"go game exited before attackers fed (xmage at T%d, go-state=%s, gameErr=%w)",
					msg.Turn, mg.snapshotState(), mg.gameErr)
			case <-abortCh:
				return run, fmt.Errorf(
					"aborted (xmage at T%d, attackers) — likely game timeout", msg.Turn)
			}
			dbg(debug, "main: attackCh sent")

			if err := src.send(ack); err != nil {
				return run, fmt.Errorf("send ack: %w", err)
			}

		case "blockers_declared":
			run.numDecisions++
			line := fmt.Sprintf("T%d blockers: %v", msg.Turn, msg.Blockers)
			run.decisions = append(run.decisions, line)
			if verbose {
				fmt.Printf("    %s\n", line)
			}
			dbg(debug, "main: -> blockCh[p%d] (n=%d) go-state=%s",
				msg.PlayerIdx, len(msg.Blockers), mg.snapshotState())
			select {
			case mg.players[msg.PlayerIdx].blockCh <- msg.Blockers:
			case <-mg.doneCh:
				return run, fmt.Errorf(
					"go game exited before blockers fed (xmage at T%d, go-state=%s, gameErr=%w)",
					msg.Turn, mg.snapshotState(), mg.gameErr)
			case <-abortCh:
				return run, fmt.Errorf(
					"aborted (xmage at T%d, blockers) — likely game timeout", msg.Turn)
			}
			dbg(debug, "main: blockCh sent")

			if err := src.send(ack); err != nil {
				return run, fmt.Errorf("send ack: %w", err)
			}

		default:
			// Ignore unknown message types
		}
	}
}

// formatActionLine renders one xmage action as a single log line. Used for both
// the in-memory decision history and verbose stdout, so they stay in sync.
func formatActionLine(turn int, step string, playerIdx int, a *actionInfo) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "T%d %s p%d: %s", turn, step, playerIdx, a.Kind)
	if a.CardName != "" {
		fmt.Fprintf(&sb, " %s", a.CardName)
	}
	if a.PermanentName != "" && a.PermanentName != a.CardName {
		fmt.Fprintf(&sb, " perm=%s", a.PermanentName)
	}
	if a.Kind == "activate_ability" {
		fmt.Fprintf(&sb, " ability=%d", a.AbilityIndex)
	}
	if len(a.Targets) > 0 {
		fmt.Fprintf(&sb, " targets=%v", a.Targets)
	}
	if a.X > 0 {
		fmt.Fprintf(&sb, " x=%d", a.X)
	}
	return sb.String()
}

// buildFirstDivergenceDump renders both engines' full view of the game at the
// moment the first stable-state divergence appeared, plus the triggering
// action and a tail of recent decisions. Format is plain text — meant for
// humans reading logs/game_NNNN_first_divergence.log.
func buildFirstDivergenceDump(msg *oracleMsg, action string, goState, xmageState *cvState, divergences, decisions []string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# First divergence at T%d %s p%d\n", msg.Turn, msg.Step, msg.PlayerIdx)
	fmt.Fprintf(&sb, "# Triggering action: %s\n\n", action)

	fmt.Fprintf(&sb, "## Divergent fields\n")
	for _, d := range divergences {
		fmt.Fprintf(&sb, "  %s\n", d)
	}

	fmt.Fprintf(&sb, "\n%s", formatStateDump("mage-go", goState))
	fmt.Fprintf(&sb, "\n%s", formatStateDump("xmage", xmageState))

	tail := decisions
	const tailLimit = 40
	if len(tail) > tailLimit {
		tail = tail[len(tail)-tailLimit:]
		fmt.Fprintf(&sb, "\n## Last %d decisions (of %d total)\n", tailLimit, len(decisions))
	} else {
		fmt.Fprintf(&sb, "\n## Decisions (%d total)\n", len(decisions))
	}
	for _, d := range tail {
		fmt.Fprintf(&sb, "  %s\n", d)
	}
	return sb.String()
}

func formatStateDump(label string, st *cvState) string {
	if st == nil {
		return fmt.Sprintf("## %s state: <nil>\n", label)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "## %s state\n", label)
	fmt.Fprintf(&sb, "  T%d %s active=%d\n", st.Turn, st.Step, st.ActivePlayerIdx)
	if len(st.Stack) > 0 {
		fmt.Fprintf(&sb, "  stack: %s\n", strings.Join(st.Stack, ", "))
	}
	for i, p := range st.Players {
		fmt.Fprintf(&sb, "  player[%d] %s: life=%d library=%d\n", i, p.Name, p.Life, p.LibrarySize)
		if len(p.Hand) > 0 {
			fmt.Fprintf(&sb, "    hand: %s\n", strings.Join(p.Hand, ", "))
		}
		if len(p.Battlefield) > 0 {
			parts := make([]string, len(p.Battlefield))
			for j, perm := range p.Battlefield {
				tag := ""
				if perm.Tapped {
					tag += "(T)"
				}
				if perm.SummonSick {
					tag += "(SS)"
				}
				parts[j] = fmt.Sprintf("%s[%d/%d]%s", perm.Name, perm.Power, perm.Toughness, tag)
			}
			fmt.Fprintf(&sb, "    battlefield: %s\n", strings.Join(parts, ", "))
		}
		if len(p.Graveyard) > 0 {
			fmt.Fprintf(&sb, "    graveyard: %s\n", strings.Join(p.Graveyard, ", "))
		}
	}
	return sb.String()
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

func writeDecisionLog(dir string, gameNum int, decisions []string) {
	if len(decisions) == 0 {
		return
	}
	fname := fmt.Sprintf("%s/game_%04d_decisions.log", dir, gameNum)
	f, err := os.Create(fname)
	if err != nil {
		return
	}
	defer f.Close()
	for _, d := range decisions {
		fmt.Fprintln(f, d)
	}
}

func writeFirstDivergenceLog(dir string, gameNum int, dump string) {
	fname := fmt.Sprintf("%s/game_%04d_first_divergence.log", dir, gameNum)
	f, err := os.Create(fname)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprint(f, dump)
}

// runReplayMode runs mage-go alone against a directory of recorded transcripts.
// No JVM, no IPC, no oracle restarts. Each transcript is self-describing
// (decks, hand size, max turns, original seed embedded in its header), so the
// only flag that matters is replayDir + the usual output controls.
//
// One transcript is one game. Files are read in lexical order; -game-only N
// selects the Nth file in that order.
func runReplayMode(replayDir, outDir string, gameOnly int, verbose, debug bool) {
	files, err := filepath.Glob(filepath.Join(replayDir, "game_*.jsonl"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list replay dir: %v\n", err)
		os.Exit(1)
	}
	sort.Strings(files)
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No transcripts found in %s (looking for game_*.jsonl)\n", replayDir)
		os.Exit(1)
	}

	if outDir != "" {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to mkdir %s: %v\n", outDir, err)
			os.Exit(1)
		}
	}

	fmt.Printf("Replay mode: %d transcripts in %s\n", len(files), replayDir)

	totalDecisions := 0
	totalWarnings := 0
	gamesOK := 0
	gamesDiverged := 0
	gamesWithWarnings := 0
	gameErrors := 0

	for i, path := range files {
		gameNum := i + 1
		if gameOnly > 0 && gameNum != gameOnly {
			continue
		}
		fmt.Printf("  Game %d/%d (%s)...\n", gameNum, len(files), filepath.Base(path))

		rs, err := newReplaySource(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Game %d open error: %v\n", gameNum, err)
			gameErrors++
			continue
		}

		// Replay never aborts on wall-clock — file IO doesn't hang. Pass a
		// nil channel so the existing <-abortCh select cases simply never fire.
		var abortCh chan struct{}
		run, err := runXMageDrivenGame(rs, rs.header.DeckA, rs.header.DeckB, rs.header.MaxTurns, verbose, debug, abortCh)
		_ = rs.close()

		stoppedAtDivergence := errors.Is(err, errStoppedAtDivergence)
		if err != nil && !stoppedAtDivergence {
			gameErrors++
			fmt.Fprintf(os.Stderr, "  Game %d error: %v\n", gameNum, err)
			if outDir != "" {
				writeErrorLog(outDir, gameNum, rs.header.Seed, rs.header.DeckA, rs.header.DeckB, err)
				writeDecisionLog(outDir, gameNum, run.decisions)
			}
			continue
		}

		totalDecisions += run.numDecisions
		totalWarnings += len(run.warnings)
		if len(run.warnings) > 0 {
			gamesWithWarnings++
		}
		if len(run.divergences) > 0 {
			gamesDiverged++
			fmt.Printf("  Game %d: %d divergences, %d warnings (out of %d decisions, stopped early=%v)\n",
				gameNum, len(run.divergences), len(run.warnings), run.numDecisions, stoppedAtDivergence)
			if verbose {
				for _, d := range run.divergences {
					fmt.Printf("    %s\n", d)
				}
			}
			if outDir != "" {
				writeGameLog(outDir, gameNum, rs.header.Seed, rs.header.DeckA, rs.header.DeckB, run.divergences, run.warnings, run.numDecisions)
				writeDecisionLog(outDir, gameNum, run.decisions)
				if run.dump != "" {
					writeFirstDivergenceLog(outDir, gameNum, run.dump)
				}
			}
		} else {
			gamesOK++
		}

		if gameOnly > 0 {
			break
		}
	}

	fmt.Printf("\n=== Replay summary ===\n")
	fmt.Printf("Games: %d, OK: %d, Diverged: %d, Errors: %d\n", len(files), gamesOK, gamesDiverged, gameErrors)
	fmt.Printf("Warnings: %d total across %d games\n", totalWarnings, gamesWithWarnings)
	fmt.Printf("Total decisions: %d\n", totalDecisions)

	if outDir != "" && (gamesDiverged > 0 || gameErrors > 0) {
		f, _ := os.Create(filepath.Join(outDir, "summary.log"))
		if f != nil {
			fmt.Fprintf(f, "replay-dir=%s files=%d\n", replayDir, len(files))
			fmt.Fprintf(f, "OK=%d diverged=%d errors=%d decisions=%d\n",
				gamesOK, gamesDiverged, gameErrors, totalDecisions)
			f.Close()
		}
		fmt.Printf("Logs written to %s/\n", outDir)
	}

	if gamesDiverged > 0 {
		os.Exit(1)
	}
}
