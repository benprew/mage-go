package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// runXMageDumpGame drives one game purely through XMage. Mage-go is not in the
// loop: there is no mirror, no priority sync, no state comparison. We just
// send setup, write every oracle message verbatim to a JSONL file, and ack.
//
// The output file is a self-describing transcript:
//   - line 1: transcript_header (schema version, seed, decks, hand size, max turns)
//   - line 2: the setup message we sent to xmage
//   - lines 3+: every oracle message verbatim, in order, ending with game_over
//
// Replay mode (replaySource) consumes this exact format.
func runXMageDumpGame(oracle *xmageOracle, deckA, deckB []string, maxTurns int, dumpPath string, seed int64, gameNum int) error {
	setup := setupMsg{
		Type:     "setup",
		PlayerA:  playerDef{Name: "Alice", Library: deckA},
		PlayerB:  playerDef{Name: "Bob", Library: deckB},
		HandSize: 7,
		MaxTurns: maxTurns,
	}
	if err := oracle.send(setup); err != nil {
		return fmt.Errorf("send setup: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dumpPath), 0o755); err != nil {
		return fmt.Errorf("mkdir dump dir: %w", err)
	}
	f, err := os.Create(dumpPath)
	if err != nil {
		return fmt.Errorf("create dump file: %w", err)
	}
	defer f.Close()

	header := transcriptHeader{
		Type:       "transcript_header",
		Version:    transcriptVersion,
		Seed:       seed,
		GameNum:    gameNum,
		DeckA:      deckA,
		DeckB:      deckB,
		HandSize:   setup.HandSize,
		MaxTurns:   maxTurns,
		RecordedAt: time.Now().UTC().Format(time.RFC3339),
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return fmt.Errorf("marshal transcript header: %w", err)
	}
	if _, err := fmt.Fprintln(f, string(headerJSON)); err != nil {
		return fmt.Errorf("write transcript header: %w", err)
	}

	setupJSON, err := json.Marshal(setup)
	if err != nil {
		return fmt.Errorf("marshal setup: %w", err)
	}
	if _, err := fmt.Fprintln(f, string(setupJSON)); err != nil {
		return fmt.Errorf("write setup: %w", err)
	}

	ack := ackMsg{Type: "ack"}

	for {
		line, msg, err := oracle.recvLine(120 * time.Second)
		if err != nil {
			return fmt.Errorf("recv: %w", err)
		}
		if _, err := fmt.Fprintln(f, line); err != nil {
			return fmt.Errorf("write line: %w", err)
		}

		switch msg.Type {
		case "game_over":
			return nil
		case "error":
			return fmt.Errorf("oracle: %s", msg.Message)
		}

		if err := oracle.send(ack); err != nil {
			return fmt.Errorf("send ack: %w", err)
		}
	}
}
