package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRulingsStoreRequiresFiles(t *testing.T) {
	dir := t.TempDir()
	names := []string{"Time Vault"}
	if err := checkRulingsStore(dir, names); err == nil || !strings.Contains(err.Error(), `go run ./cmd/fetchset -card "Time Vault"`) {
		t.Fatalf("missing index must give a download command: %v", err)
	}
	card := Card{Name: names[0], OracleID: "99d4d99d-cf56-45aa-aa39-a250695612f2", Rulings: []Ruling{}}
	if err := writeRulingsStore(dir, []Card{card}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := checkRulingsStore(dir, names); err != nil {
		t.Fatalf("explicit empty rulings must pass: %v", err)
	}
	path := filepath.Join(dir, card.OracleID+".json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := checkRulingsStore(dir, names); err == nil || !strings.Contains(err.Error(), names[0]) || !strings.Contains(err.Error(), "-card") {
		t.Fatalf("missing file must identify the card and download command: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"name":"Time Vault","rulings":null}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkRulingsStore(dir, names); err == nil {
		t.Fatal("invalid ruling record accepted")
	}
}

func TestRONCollectionHasRulings(t *testing.T) {
	names, err := ronCardNames("../../cards/mtgish")
	if err != nil {
		t.Fatal(err)
	}
	if err := checkRulingsStore("../../data/rulings", names); err != nil {
		t.Fatal(err)
	}
}

func TestRulingsStoreMergeAndRefresh(t *testing.T) {
	dir := t.TempDir()
	stamp := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	first := Card{Name: "Time Vault", OracleID: "99d4d99d-cf56-45aa-aa39-a250695612f2", RulingsURI: "https://example.com/rulings", Rulings: []Ruling{{Source: "wotc", Comment: "Original"}}}
	if err := writeRulingsStore(dir, []Card{first, first}, stamp); err != nil {
		t.Fatal(err)
	}
	second := Card{Name: "Island", OracleID: "00000000-0000-0000-0000-000000000001", Rulings: []Ruling{}}
	if err := writeRulingsStore(dir, []Card{second}, stamp); err != nil {
		t.Fatal(err)
	}
	var index map[string]string
	readJSON(t, filepath.Join(dir, "index.json"), &index)
	if len(index) != 2 || index[first.Name] != first.OracleID+".json" {
		t.Fatalf("index lost entries: %v", index)
	}
	first.Rulings[0].Comment = "Refreshed"
	if err := writeRulingsStore(dir, []Card{first}, stamp.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	var saved storedRulings
	readJSON(t, filepath.Join(dir, index[first.Name]), &saved)
	if saved.Rulings[0].Comment != "Refreshed" || saved.RetrievedAt != stamp.Add(time.Hour).Format(time.RFC3339) || saved.Name != first.Name {
		t.Fatalf("refresh failed: %+v", saved)
	}
	readJSON(t, filepath.Join(dir, index[second.Name]), &saved)
	if saved.Rulings == nil || len(saved.Rulings) != 0 {
		t.Fatal("empty rulings must remain explicit")
	}
}

func TestRulingsStoreRejectsInvalidID(t *testing.T) {
	dir := t.TempDir()
	if err := writeRulingsStore(dir, []Card{{Name: "Bad", OracleID: "../escape"}}, time.Now()); err == nil {
		t.Fatal("invalid Oracle ID accepted")
	}
}

func readJSON(t *testing.T, path string, dst any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		t.Fatal(err)
	}
}
