package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchSetWithRulings(t *testing.T) {
	requests := make(map[string]int)
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		if r.Header.Get("User-Agent") == "" || r.Header.Get("Accept") != "application/json" {
			t.Error("missing API headers")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/cards":
			fmtJSON(t, w, map[string]any{"data": []map[string]string{
				{"name": "Time Vault", "oracle_id": "vault", "rulings_uri": server.URL + "/rulings"},
			}, "has_more": true, "next_page": server.URL + "/page2"})
		case "/page2":
			fmtJSON(t, w, map[string]any{"data": []map[string]string{
				{"name": "Time Vault", "oracle_id": "vault", "rulings_uri": server.URL + "/duplicate"},
				{"name": "Island", "oracle_id": "island", "rulings_uri": server.URL + "/empty"},
			}})
		case "/rulings":
			fmtJSON(t, w, map[string]any{"data": []map[string]string{{"oracle_id": "vault", "source": "wotc", "published_at": "2008-10-01", "comment": "First note."}}, "has_more": true, "next_page": server.URL + "/rulings2"})
		case "/rulings2":
			fmtJSON(t, w, map[string]any{"data": []map[string]string{{"oracle_id": "vault", "source": "scryfall", "published_at": "2026-01-01", "comment": "Second note."}}})
		case "/empty":
			_, _ = w.Write([]byte(`{"data":[]}`))
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	cards, err := fetchSetWithClient(server.Client(), server.URL+"/cards", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 3 || len(cards[0].Rulings) != 2 || len(cards[1].Rulings) != 2 {
		t.Fatalf("missing cards or rulings: %+v", cards)
	}
	if cards[0].Rulings[1].Source != "scryfall" || cards[0].Rulings[0].PublishedAt != "2008-10-01" || cards[0].Rulings[0].Comment != "First note." {
		t.Fatalf("ruling fields changed: %+v", cards[0].Rulings)
	}
	if requests["/rulings"] != 1 || requests["/duplicate"] != 0 {
		t.Fatal("duplicate Oracle card fetched again")
	}
	encoded, err := json.Marshal(cards[2])
	if err != nil || !strings.Contains(string(encoded), `"rulings":[]`) {
		t.Fatalf("empty rulings must be explicit: %s, %v", encoded, err)
	}
}

func TestFetchSetRulingFailure(t *testing.T) {
	for _, response := range []string{"http-error", "invalid-json", "missing-uri", "missing-next-page"} {
		t.Run(response, func(t *testing.T) {
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/cards" {
					uri := server.URL + "/rulings"
					if response == "missing-uri" {
						uri = ""
					}
					fmtJSON(t, w, map[string]any{"data": []map[string]string{{"name": "Time Vault", "rulings_uri": uri}}})
					return
				}
				switch response {
				case "http-error":
					w.WriteHeader(http.StatusTooManyRequests)
					_, _ = w.Write([]byte(`{"details":"slow down"}`))
				case "missing-next-page":
					_, _ = w.Write([]byte(`{"data":[],"has_more":true}`))
				default:
					_, _ = w.Write([]byte(`invalid`))
				}
			}))
			defer server.Close()
			if _, err := fetchSetWithClient(server.Client(), server.URL+"/cards", 0); err == nil {
				t.Fatal("incomplete ruling download succeeded")
			}
		})
	}
}

func fmtJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Error(err)
	}
}
