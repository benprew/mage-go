// fetchset fetches all cards in an MTG set from the Scryfall API and writes
// a JSON array to stdout.
//
// Usage:
//
//	go run ./cmd/fetchset ARN       # Arabian Nights
//	go run ./cmd/fetchset LEA       # Alpha
//	go run ./cmd/fetchset -o out.json ARN
package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Card holds the fields relevant to the mage engine.
type Card struct {
	Name       string   `json:"name"`
	ManaCost   string   `json:"mana_cost,omitempty"`
	CMC        float64  `json:"cmc"`
	TypeLine   string   `json:"type_line"`
	OracleText string   `json:"oracle_text,omitempty"`
	Power      string   `json:"power,omitempty"`
	Toughness  string   `json:"toughness,omitempty"`
	Loyalty    string   `json:"loyalty,omitempty"`
	Colors     []string `json:"colors"`
	Keywords   []string `json:"keywords"`
	Rarity     string   `json:"rarity"`
	Set        string   `json:"set"`
	// Card faces for double-faced cards (DFCs)
	CardFaces []CardFace `json:"card_faces,omitempty"`
}

type CardFace struct {
	Name       string   `json:"name"`
	ManaCost   string   `json:"mana_cost,omitempty"`
	TypeLine   string   `json:"type_line,omitempty"`
	OracleText string   `json:"oracle_text,omitempty"`
	Power      string   `json:"power,omitempty"`
	Toughness  string   `json:"toughness,omitempty"`
	Loyalty    string   `json:"loyalty,omitempty"`
	Colors     []string `json:"colors,omitempty"`
}

type scryfallResponse struct {
	Data     []Card   `json:"data"`
	HasMore  bool     `json:"has_more"`
	NextPage string   `json:"next_page"`
	Warnings []string `json:"warnings"`
	// error fields
	Status  int    `json:"status"`
	Details string `json:"details"`
}

func main() {
	outFile := flag.String("o", "", "write output to file instead of stdout")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: fetchset [flags] <set-code>\n\n")
		fmt.Fprintf(os.Stderr, "Fetches all cards in a set from Scryfall and writes JSON to stdout.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	setCode := flag.Arg(0)

	cards, err := fetchSet(setCode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	out, err := json.MarshalIndent(cards, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
		os.Exit(1)
	}

	if *outFile != "" {
		if err := os.WriteFile(*outFile, out, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote %d cards to %s\n", len(cards), *outFile)
	} else {
		fmt.Println(string(out))
	}
}

func fetchSet(setCode string) ([]Card, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if os.Getenv("FETCHSET_SKIP_TLS") != "" {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // sandbox workaround
	}
	client := &http.Client{Timeout: 15 * time.Second, Transport: transport}
	// unique=prints would include reprints; unique=cards deduplicates by name.
	// For a set list we want every card slot, so use unique=prints.
	url := fmt.Sprintf(
		"https://api.scryfall.com/cards/search?q=set:%s&order=set&unique=prints",
		setCode,
	)

	var all []Card
	for url != "" {
		var page scryfallResponse
		if err := get(client, url, &page); err != nil {
			return nil, err
		}
		if page.Status != 0 {
			// Scryfall error response
			return nil, fmt.Errorf("scryfall: %s (status %d)", page.Details, page.Status)
		}
		all = append(all, page.Data...)
		fmt.Fprintf(os.Stderr, "fetched %d / %d+ cards...\n", len(all), len(all))

		if page.HasMore {
			url = page.NextPage
			time.Sleep(80 * time.Millisecond) // stay within Scryfall's 10 req/sec limit
		} else {
			url = ""
		}
	}
	return all, nil
}

func get(client *http.Client, url string, dst any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("User-Agent", "mage-go/fetchset (git.sr.ht/~cdcarter/mage-go)")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	return json.Unmarshal(body, dst)
}
