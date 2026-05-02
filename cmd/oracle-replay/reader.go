package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// recordingReader streams a JSONL recording one event at a time.
// The first line is META; subsequent lines are EVENT.
//
// Use scanLargeBuffer because XMage snapshots include full hand+battlefield
// per event and a single line can easily exceed bufio.Scanner's default
// 64KB. We size at 16MB to comfortably swallow the largest snapshots seen
// in practice.
const scanLargeBuffer = 16 * 1024 * 1024

type recordingReader struct {
	src     io.Closer  // file or gzip wrapper to close
	scanner *bufio.Scanner
	meta    metaLine
	headRead bool
}

func openRecording(path string) (*recordingReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	var rdr io.Reader = f
	closer := io.Closer(f)
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("gzip: %w", err)
		}
		rdr = gz
		closer = closeBoth{gz: gz, f: f}
	}

	scanner := bufio.NewScanner(rdr)
	scanner.Buffer(make([]byte, 0, 64*1024), scanLargeBuffer)

	rr := &recordingReader{src: closer, scanner: scanner}
	if err := rr.readMeta(); err != nil {
		closer.Close()
		return nil, fmt.Errorf("read meta: %w", err)
	}
	return rr, nil
}

func (r *recordingReader) readMeta() error {
	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return err
		}
		return fmt.Errorf("empty file (no META line)")
	}
	if err := json.Unmarshal(r.scanner.Bytes(), &r.meta); err != nil {
		return fmt.Errorf("decode META: %w", err)
	}
	if r.meta.Record != "META" {
		return fmt.Errorf("first line is not META (got record=%q)", r.meta.Record)
	}
	r.headRead = true
	return nil
}

// Meta returns the parsed META line. Valid after openRecording succeeds.
func (r *recordingReader) Meta() metaLine { return r.meta }

// Next reads the next EVENT line. Returns io.EOF after the last event.
// Lines that don't decode as EVENT (e.g. unknown record type) are skipped
// silently — keeps the reader forward-compatible with new record types.
func (r *recordingReader) Next() (*eventLine, error) {
	for r.scanner.Scan() {
		var ev eventLine
		if err := json.Unmarshal(r.scanner.Bytes(), &ev); err != nil {
			return nil, fmt.Errorf("decode event: %w", err)
		}
		if ev.Record != "EVENT" {
			continue
		}
		return &ev, nil
	}
	if err := r.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

func (r *recordingReader) Close() error { return r.src.Close() }

// closeBoth chains gzip-reader and underlying-file closes so callers get
// one Close that releases both fds.
type closeBoth struct {
	gz *gzip.Reader
	f  *os.File
}

func (c closeBoth) Close() error {
	gzErr := c.gz.Close()
	fErr := c.f.Close()
	if gzErr != nil {
		return gzErr
	}
	return fErr
}
