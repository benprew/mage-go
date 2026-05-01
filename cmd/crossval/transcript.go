package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// transcriptVersion bumps when the on-disk format changes incompatibly.
// Replay refuses to read transcripts from a future version.
const transcriptVersion = 1

// transcriptHeader is the first JSONL record in a recorded game. It captures
// the deck/seed/turn metadata alongside the wire stream so a transcript is
// self-describing — replay reads everything it needs from the file.
type transcriptHeader struct {
	Type       string   `json:"type"` // always "transcript_header"
	Version    int      `json:"version"`
	Seed       int64    `json:"seed"`
	GameNum    int      `json:"game_num"`
	DeckA      []string `json:"deck_a"`
	DeckB      []string `json:"deck_b"`
	HandSize   int      `json:"hand_size"`
	MaxTurns   int      `json:"max_turns"`
	RecordedAt string   `json:"recorded_at"` // RFC3339
}

// eventSource is the minimal interface runXMageDrivenGame needs from "the
// other engine": somewhere to send setup/ack/card-check, and somewhere to recv
// the next oracle message. The live xmageOracle and the replaySource both
// satisfy it; everything else (process management, stderr capture, restart on
// crash) is handled in main.go branched by mode.
type eventSource interface {
	send(any) error
	recv(timeout time.Duration) (*oracleMsg, error)
}

// replaySource feeds a recorded JSONL transcript into runXMageDrivenGame as
// if it had come from a live xmage. send is a no-op: there's no counterparty
// to talk back to.
type replaySource struct {
	path   string
	f      *os.File
	dec    *json.Decoder
	header transcriptHeader
	setup  setupMsg
	closed bool
}

// newReplaySource opens a transcript file and consumes the header + setup
// records, leaving the decoder positioned on the first oracle event.
func newReplaySource(path string) (*replaySource, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open transcript %s: %w", path, err)
	}
	dec := json.NewDecoder(bufio.NewReader(f))

	var h transcriptHeader
	if err := dec.Decode(&h); err != nil {
		f.Close()
		return nil, fmt.Errorf("read transcript header from %s: %w", path, err)
	}
	if h.Type != "transcript_header" {
		f.Close()
		return nil, fmt.Errorf("%s: first record is not a transcript_header (got type=%q)", path, h.Type)
	}
	if h.Version != transcriptVersion {
		f.Close()
		return nil, fmt.Errorf("%s: transcript version %d, this binary speaks %d", path, h.Version, transcriptVersion)
	}

	var setup setupMsg
	if err := dec.Decode(&setup); err != nil {
		f.Close()
		return nil, fmt.Errorf("%s: read setup record: %w", path, err)
	}
	if setup.Type != "setup" {
		f.Close()
		return nil, fmt.Errorf("%s: second record is not a setup (got type=%q)", path, setup.Type)
	}

	return &replaySource{
		path:   path,
		f:      f,
		dec:    dec,
		header: h,
		setup:  setup,
	}, nil
}

func (rs *replaySource) send(any) error { return nil }

func (rs *replaySource) recv(_ time.Duration) (*oracleMsg, error) {
	if rs.closed {
		return nil, io.EOF
	}
	var m oracleMsg
	if err := rs.dec.Decode(&m); err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("transcript %s ended before game_over", rs.path)
		}
		return nil, fmt.Errorf("decode transcript event in %s: %w", rs.path, err)
	}
	return &m, nil
}

func (rs *replaySource) close() error {
	if rs == nil || rs.closed {
		return nil
	}
	rs.closed = true
	return rs.f.Close()
}
