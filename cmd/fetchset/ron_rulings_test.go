package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeBulkRulings(t *testing.T) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, _ = writer.Write([]byte("{\"oracle_id\":\"vault\",\"comment\":\"First\"}\n{\"oracle_id\":\"vault\",\"comment\":\"Second\"}\n"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{compressed.Bytes(), []byte(`[{"oracle_id":"vault","comment":"First"},{"oracle_id":"vault","comment":"Second"}]`)} {
		rulings, err := decodeBulkRulings(data)
		if err != nil || len(rulings) != 2 || rulings[1].Comment != "Second" {
			t.Fatalf("rulings = %+v, %v", rulings, err)
		}
	}
	if _, err := decodeBulkRulings([]byte(`invalid`)); err == nil {
		t.Fatal("invalid bulk data accepted")
	}
}

func TestRONCardNames(t *testing.T) {
	dir := t.TempDir()
	for _, file := range []string{"A.ron", "B.ron"} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(`Card(Name:"Time Vault",Rules:[])`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	names, err := ronCardNames(dir)
	if err != nil || len(names) != 1 || names[0] != "Time Vault" {
		t.Fatalf("names = %v, %v", names, err)
	}
	if _, err := ronCardNames(t.TempDir()); err == nil {
		t.Fatal("empty directory accepted")
	}
}

func TestCollectionRulings(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete", true: "missing"}[missing], func(t *testing.T) {
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/cards/collection":
					var body struct {
						Identifiers []map[string]string `json:"identifiers"`
					}
					if r.Method != http.MethodPost || json.NewDecoder(r.Body).Decode(&body) != nil || len(body.Identifiers) != 2 {
						t.Error("invalid collection request")
					}
					if missing {
						fmtJSON(t, w, map[string]any{"not_found": body.Identifiers})
						return
					}
					fmtJSON(t, w, map[string]any{"data": []Card{{Name: "Time Vault", OracleID: "vault"}, {Name: "Island", OracleID: "island"}}})
				case "/bulk-data/rulings":
					fmtJSON(t, w, map[string]string{"download_uri": server.URL + "/download"})
				case "/download":
					fmtJSON(t, w, []Ruling{{OracleID: "vault", Source: "wotc", Comment: "Note"}, {OracleID: "unrelated", Comment: "Other"}})
				default:
					t.Errorf("unexpected URL %s", r.URL)
				}
			}))
			defer server.Close()
			cards, err := fetchCollectionRulings(server.Client(), server.URL, []string{"Time Vault", "Island"}, 0)
			if missing {
				if err == nil {
					t.Fatal("missing lookup accepted")
				}
				return
			}
			if err != nil || len(cards) != 2 || len(cards[0].Rulings) != 1 || cards[1].Rulings == nil || len(cards[1].Rulings) != 0 {
				t.Fatalf("cards = %+v, %v", cards, err)
			}
		})
	}
}
