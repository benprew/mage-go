// Command scenarioanalyze reads JSONL output from scenariotest and flags
// potential engine bugs.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/benprew/mage-go/internal/scenario"
)

func main() {
	input := flag.String("input", "", "JSONL file to analyze (default: stdin)")
	verbose := flag.Bool("verbose", false, "show details for each flagged game")
	flag.Parse()

	var scanner *bufio.Scanner
	if *input != "" {
		f, err := os.Open(*input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer func() { _ = f.Close() }()
		scanner = bufio.NewScanner(f)
	} else {
		scanner = bufio.NewScanner(os.Stdin)
	}
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var results []scenario.GameResult
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		var r scenario.GameResult
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			fmt.Fprintf(os.Stderr, "warning: line %d: %v\n", lineNum, err)
			continue
		}
		results = append(results, r)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("No games to analyze.")
		return
	}

	analyze(results, *verbose)
}

func analyze(results []scenario.GameResult, verbose bool) {
	total := len(results)

	// Categorize games.
	var panics, maxTurns, timeouts []scenario.GameResult
	var withAnomalies []scenario.GameResult
	var turns []int
	panicGroups := map[string][]scenario.GameResult{}
	winCounts := map[string]int{}
	deckWins := map[string]int{}
	deckGames := map[string]int{}
	var shortGames []scenario.GameResult

	for _, r := range results {
		turns = append(turns, r.FinalTurn)

		switch r.WinReason {
		case "panic":
			panics = append(panics, r)
			key := firstLine(r.PanicMsg)
			panicGroups[key] = append(panicGroups[key], r)
		case "max_turns":
			maxTurns = append(maxTurns, r)
		case "timeout":
			timeouts = append(timeouts, r)
		}

		if len(r.Anomalies) > 0 {
			withAnomalies = append(withAnomalies, r)
		}
		if r.Winner != "" {
			winCounts[r.Winner]++
		}
		if r.FinalTurn <= 2 && r.WinReason != "panic" {
			shortGames = append(shortGames, r)
		}

		deckGames[r.DeckA]++
		deckGames[r.DeckB]++
		switch r.Winner {
		case "Alice":
			deckWins[r.DeckA]++
		case "Bob":
			deckWins[r.DeckB]++
		}
	}

	sort.Ints(turns)

	fmt.Printf("Scenario Test Analysis (%d games)\n", total)
	fmt.Println(strings.Repeat("-", 45))

	// Panics.
	fmt.Printf("Panics:             %d (%.1f%%)\n", len(panics), pct(len(panics), total))
	if len(panics) > 0 {
		type panicEntry struct {
			msg   string
			count int
			games []scenario.GameResult
		}
		var entries []panicEntry
		for msg, games := range panicGroups {
			entries = append(entries, panicEntry{msg, len(games), games})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].count > entries[j].count })
		for _, e := range entries {
			fmt.Printf("  %q — %d game(s)\n", truncate(e.msg, 80), e.count)
			if verbose {
				for _, g := range e.games {
					fmt.Printf("    game %d: %s vs %s (turn %d)\n", g.GameID, g.DeckA, g.DeckB, g.FinalTurn)
				}
			}
		}
	}

	// Max turns.
	fmt.Printf("Max turns reached:  %d (%.1f%%)\n", len(maxTurns), pct(len(maxTurns), total))
	if verbose && len(maxTurns) > 0 {
		for _, g := range maxTurns {
			fmt.Printf("  game %d: %s vs %s (life: %d/%d)\n", g.GameID, g.DeckA, g.DeckB, g.FinalLifeA, g.FinalLifeB)
		}
	}

	// Timeouts.
	fmt.Printf("Timeouts:           %d (%.1f%%)\n", len(timeouts), pct(len(timeouts), total))
	if verbose && len(timeouts) > 0 {
		for _, g := range timeouts {
			fmt.Printf("  game %d: %s vs %s (turn %d)\n", g.GameID, g.DeckA, g.DeckB, g.FinalTurn)
		}
	}

	// Anomalies.
	fmt.Printf("Anomalies:          %d (%.1f%%)\n", len(withAnomalies), pct(len(withAnomalies), total))
	if len(withAnomalies) > 0 {
		anomalyCounts := map[string]int{}
		for _, g := range withAnomalies {
			for _, a := range g.Anomalies {
				key, _, _ := strings.Cut(a, ":")
				anomalyCounts[key]++
			}
		}
		for key, count := range anomalyCounts {
			fmt.Printf("  %s: %d\n", key, count)
		}
		if verbose {
			for _, g := range withAnomalies {
				fmt.Printf("  game %d: %s vs %s — %v\n", g.GameID, g.DeckA, g.DeckB, g.Anomalies)
			}
		}
	}

	// Short games.
	fmt.Printf("Very short (≤2t):   %d (%.1f%%)\n", len(shortGames), pct(len(shortGames), total))
	if verbose && len(shortGames) > 0 {
		for _, g := range shortGames {
			fmt.Printf("  game %d: %s vs %s → %s (turn %d, life: %d/%d)\n",
				g.GameID, g.DeckA, g.DeckB, g.Winner, g.FinalTurn, g.FinalLifeA, g.FinalLifeB)
		}
	}

	// Game length stats.
	fmt.Println()
	fmt.Printf("Game length:        median=%d  p95=%d  min=%d  max=%d\n",
		percentile(turns, 50), percentile(turns, 95), turns[0], turns[len(turns)-1])

	// Win rates.
	completed := 0
	for _, c := range winCounts {
		completed += c
	}
	fmt.Printf("Win rate:           Alice=%d (%.0f%%)  Bob=%d (%.0f%%)",
		winCounts["Alice"], pct(winCounts["Alice"], total),
		winCounts["Bob"], pct(winCounts["Bob"], total))
	draws := total - completed - len(panics) - len(timeouts)
	if draws > 0 {
		fmt.Printf("  draws/stalls=%d (%.0f%%)", draws, pct(draws, total))
	}
	fmt.Println()

	// Deck win rates (sorted by games played).
	type deckStat struct {
		name  string
		games int
		wins  int
	}
	var deckStats []deckStat
	for name, games := range deckGames {
		deckStats = append(deckStats, deckStat{name, games, deckWins[name]})
	}
	sort.Slice(deckStats, func(i, j int) bool { return deckStats[i].games > deckStats[j].games })

	fmt.Println()
	fmt.Println("Deck performance:")
	for _, ds := range deckStats {
		winRate := 0.0
		if ds.games > 0 {
			winRate = float64(ds.wins) / float64(ds.games) * 100
		}
		fmt.Printf("  %-30s %2d games  %2d wins  (%.0f%%)\n", ds.name, ds.games, ds.wins, winRate)
	}

	// Skipped cards summary.
	skippedSet := map[string]bool{}
	for _, r := range results {
		for _, c := range r.CardsSkippedA {
			skippedSet[c] = true
		}
		for _, c := range r.CardsSkippedB {
			skippedSet[c] = true
		}
	}
	if len(skippedSet) > 0 {
		var skippedList []string
		for c := range skippedSet {
			skippedList = append(skippedList, c)
		}
		sort.Strings(skippedList)
		fmt.Printf("\nUnregistered cards encountered (%d):\n", len(skippedList))
		for _, c := range skippedList {
			fmt.Printf("  %s\n", c)
		}
	}
}

func pct(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total) * 100
}

func percentile(sorted []int, p int) int {
	if len(sorted) == 0 {
		return 0
	}
	idx := max(int(math.Ceil(float64(p)/100*float64(len(sorted))))-1, 0)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func firstLine(s string) string {
	if before, _, ok := strings.Cut(s, "\n"); ok {
		return before
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
