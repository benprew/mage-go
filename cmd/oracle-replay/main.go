// oracle-replay reads an XMage SelfPlayJsonRecorder JSONL file and replays
// the game in mage-go, validating state at every recorded priority point.
//
// Usage:
//
//	oracle-replay path/to/game.jsonl[.gz]
//
// Exit codes:
//
//	0 — game replayed cleanly, every PRIORITY snapshot matched
//	1 — divergence detected (single first-failure mode); diff printed to stderr
//	2 — usage / file / parse error
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	_ "github.com/benprew/mage-go/cards" // register all sets
	"github.com/benprew/mage-go/pkg/mage"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		loose   bool
		verbose bool
	)
	flag.BoolVar(&loose, "loose", true, "loose action-set match (kind+sourceName), ignoring text")
	flag.BoolVar(&verbose, "verbose", false, "print one line per PRIORITY validation")
	var trace bool
	flag.BoolVar(&trace, "trace", false, "enable mage-go priority-action debug trace (DebugPriority)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] <recording.jsonl[.gz]>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		return 2
	}
	if trace {
		mage.DebugPriority = true
	}
	path := flag.Arg(0)

	rdr, err := openRecording(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open recording: %v\n", err)
		return 2
	}
	defer rdr.Close()

	meta := rdr.Meta()
	fmt.Printf("Replaying %s\n  game=%s started=%s players=%s vs %s totalEvents=%d\n",
		path, meta.GameID, meta.StartedAt, meta.Players[0].Name, meta.Players[1].Name, meta.TotalEvents)

	events := make([]eventLine, 0, meta.TotalEvents)
	for {
		ev, err := rdr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "read events: %v\n", err)
			return 2
		}
		events = append(events, *ev)
	}
	fmt.Printf("  loaded %d events\n", len(events))

	r, err := newReplay(meta, events, loose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build replay: %v\n", err)
		return 2
	}

	err = r.run()
	if len(r.warnings) > 0 {
		fmt.Printf("  %d action-set warning(s)\n", len(r.warnings))
		if verbose {
			for _, w := range r.warnings {
				fmt.Fprintf(os.Stderr, "  WARN %s\n", w)
			}
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "DIVERGENCE: %v\n", err)
		return 1
	}

	fmt.Println("OK — replay completed cleanly")
	return 0
}
