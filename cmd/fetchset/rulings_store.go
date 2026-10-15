package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type storedRulings struct {
	Name        string   `json:"name"`
	OracleID    string   `json:"oracle_id"`
	RulingsURI  string   `json:"rulings_uri"`
	RetrievedAt string   `json:"retrieved_at"`
	Rulings     []Ruling `json:"rulings"`
}

func checkRulingsStore(dir string, names []string) error {
	if len(names) == 0 {
		return fmt.Errorf("no card names to check")
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	index := make(map[string]string)
	data, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err == nil {
		err = json.Unmarshal(data, &index)
	}
	if err != nil || index == nil {
		return fmt.Errorf("rulings index is missing or invalid in %s; download with go run ./cmd/fetchset -card %q -rulings-dir %q, or refresh all with go run ./cmd/fetchset -ron cards/mtgish -rulings-dir %q", dir, names[0], dir, dir)
	}
	var failures []error
	for _, name := range names {
		file, exists := index[name]
		if !exists || filepath.Base(file) != file || filepath.Ext(file) != ".json" {
			failures = append(failures, fmt.Errorf("rulings entry missing or invalid for %q; download with go run ./cmd/fetchset -card %q -rulings-dir %q", name, name, dir))
			continue
		}
		var record storedRulings
		data, err := os.ReadFile(filepath.Join(dir, file))
		if err == nil {
			err = json.Unmarshal(data, &record)
		}
		if err != nil || record.Name != name || file != record.OracleID+".json" || record.RetrievedAt == "" || record.Rulings == nil {
			failures = append(failures, fmt.Errorf("rulings file missing or invalid for %q at %s; download with go run ./cmd/fetchset -card %q -rulings-dir %q", name, filepath.Join(dir, file), name, dir))
		}
	}
	return errors.Join(failures...)
}

func writeRulingsStore(dir string, cards []Card, retrieved time.Time) error {
	index := make(map[string]string)
	data, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err == nil {
		if err := json.Unmarshal(data, &index); err != nil {
			return fmt.Errorf("read rulings index: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if index == nil {
		return fmt.Errorf("rulings index must be a JSON object")
	}
	for _, card := range cards {
		if _, err := uuid.Parse(card.OracleID); err != nil || card.Name == "" || len(card.OracleID) != 36 {
			return fmt.Errorf("%s: invalid card name or Oracle ID %q", card.Name, card.OracleID)
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, card := range cards {
		rulings := card.Rulings
		if rulings == nil {
			rulings = []Ruling{}
		}
		file := card.OracleID + ".json"
		record := storedRulings{card.Name, card.OracleID, card.RulingsURI, retrieved.UTC().Format(time.RFC3339), rulings}
		if err := writeJSONAtomic(filepath.Join(dir, file), record); err != nil {
			return err
		}
		index[card.Name] = file
	}
	return writeJSONAtomic(filepath.Join(dir, "index.json"), index)
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".rulings-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0o644); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
