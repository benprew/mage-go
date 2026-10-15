// fetchset downloads cards and their rulings from the Scryfall API.
// It writes a JSON array to stdout or a file.
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
	neturl "net/url"
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
	OracleID   string   `json:"oracle_id,omitempty"`
	RulingsURI string   `json:"rulings_uri,omitempty"`
	Rulings    []Ruling `json:"rulings"`
	// Card faces for double-faced cards (DFCs)
	CardFaces []CardFace `json:"card_faces,omitempty"`
}

// Ruling keeps the source and date with each rules note.
type Ruling struct {
	Object      string `json:"object,omitempty"`
	OracleID    string `json:"oracle_id"`
	Source      string `json:"source"`
	PublishedAt string `json:"published_at"`
	Comment     string `json:"comment"`
}

type rulingsResponse struct {
	Data     []Ruling `json:"data"`
	HasMore  bool     `json:"has_more"`
	NextPage string   `json:"next_page"`
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
	rulingsDir := flag.String("rulings-dir", "data/rulings", "shared rulings directory")
	ronDir := flag.String("ron", "", "refresh only the shared rulings store for all cards in this RON directory")
	cardName := flag.String("card", "", "refresh only the shared rulings store for one card name")
	check := flag.Bool("check", false, "check local rulings without downloading; use with -ron or -card")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: fetchset [flags] <set-code>\n\n")
		fmt.Fprintf(os.Stderr, "Downloads cards and rulings from Scryfall and writes JSON to stdout.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *check {
		if flag.NArg() != 0 || *outFile != "" || ((*ronDir == "") == (*cardName == "")) {
			fmt.Fprintln(os.Stderr, "use -check with exactly one of -ron or -card, without a set code or -o")
			os.Exit(1)
		}
		names := []string{*cardName}
		var err error
		if *ronDir != "" {
			names, err = ronCardNames(*ronDir)
		}
		if err == nil {
			err = checkRulingsStore(*rulingsDir, names)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "checked rulings for %d cards\n", len(names))
		return
	}
	if *ronDir != "" || *cardName != "" {
		if flag.NArg() != 0 || (*ronDir != "" && *cardName != "") || *outFile != "" {
			fmt.Fprintln(os.Stderr, "use -ron or -card without a set code or -o")
			os.Exit(1)
		}
		client := &http.Client{Timeout: 2 * time.Minute}
		var cards []Card
		var err error
		if *ronDir != "" {
			cards, err = fetchRONRulings(client, *ronDir)
		} else {
			var card Card
			err = get(client, "https://api.scryfall.com/cards/named?exact="+neturl.QueryEscape(*cardName), &card)
			if err == nil {
				time.Sleep(110 * time.Millisecond)
				card.Rulings, err = downloadRulings(card.RulingsURI, func(uri string, dst any) error {
					err := get(client, uri, dst)
					time.Sleep(110 * time.Millisecond)
					return err
				})
				cards = []Card{card}
			}
		}
		if err == nil {
			err = writeRulingsStore(*rulingsDir, cards, time.Now())
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "stored rulings for %d cards in %s\n", len(cards), *rulingsDir)
		return
	}

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
	if err := writeRulingsStore(*rulingsDir, cards, time.Now()); err != nil {
		fmt.Fprintf(os.Stderr, "error writing rulings: %v\n", err)
		os.Exit(1)
	}

	out, err := json.MarshalIndent(cards, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
		os.Exit(1)
	}

	if *outFile != "" {
		if err := os.WriteFile(*outFile, out, 0o644); err != nil {
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
	return fetchSetWithClient(client, url, 110*time.Millisecond)
}

func fetchSetWithClient(client *http.Client, url string, interval time.Duration) ([]Card, error) {
	var lastRequest time.Time
	request := func(url string, dst any) error {
		if !lastRequest.IsZero() {
			time.Sleep(max(0, interval-time.Since(lastRequest)))
		}
		lastRequest = time.Now()
		return get(client, url, dst)
	}
	var all []Card
	for url != "" {
		var page scryfallResponse
		if err := request(url, &page); err != nil {
			return nil, err
		}
		if page.Status != 0 {
			// Scryfall error response
			return nil, fmt.Errorf("scryfall: %s (status %d)", page.Details, page.Status)
		}
		all = append(all, page.Data...)
		fmt.Fprintf(os.Stderr, "fetched %d / %d+ cards...\n", len(all), len(all))

		if page.HasMore {
			if page.NextPage == "" {
				return nil, fmt.Errorf("card page has_more without next_page")
			}
			url = page.NextPage
		} else {
			url = ""
		}
	}
	cache := make(map[string][]Ruling)
	for i := range all {
		card := &all[i]
		key := card.OracleID
		if key == "" {
			key = card.RulingsURI
		}
		if cached, ok := cache[key]; ok {
			card.Rulings = cached
			continue
		}
		if card.RulingsURI == "" {
			return nil, fmt.Errorf("%s: missing rulings_uri", card.Name)
		}
		rulings, err := downloadRulings(card.RulingsURI, request)
		if err != nil {
			return nil, fmt.Errorf("%s rulings: %w", card.Name, err)
		}
		cache[key] = rulings
		card.Rulings = rulings
	}
	return all, nil
}

func downloadRulings(uri string, request func(string, any) error) ([]Ruling, error) {
	if uri == "" {
		return nil, fmt.Errorf("missing rulings_uri")
	}
	rulings := make([]Ruling, 0)
	for uri != "" {
		var page rulingsResponse
		if err := request(uri, &page); err != nil {
			return nil, err
		}
		rulings = append(rulings, page.Data...)
		uri = ""
		if page.HasMore {
			if page.NextPage == "" {
				return nil, fmt.Errorf("rulings page has_more without next_page")
			}
			uri = page.NextPage
		}
	}
	return rulings, nil
}

func get(client *http.Client, url string, dst any) error {
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	return requestJSON(client, req, dst)
}

func requestJSON(client *http.Client, req *http.Request, dst any) error {
	body, err := requestBytes(client, req)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dst)
}

func requestBytes(client *http.Client, req *http.Request) ([]byte, error) {
	req.Header.Set("User-Agent", "mage-go/fetchset (github.com/benprew/mage-go)")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", req.Method, req.URL, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s %s: HTTP %d: %s", req.Method, req.URL, resp.StatusCode, body)
	}
	return body, nil
}
