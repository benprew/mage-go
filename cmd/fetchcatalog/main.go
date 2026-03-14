// fetchcatalog fetches card metadata from the Scryfall API for all sets
// from Alpha through Onslaught block and writes per-set JSON files to
// data/catalog/.
//
// Usage:
//
//	go run ./cmd/fetchcatalog                    # fetch all 37 sets
//	go run ./cmd/fetchcatalog -sets LEA,ARN      # fetch specific sets
//	go run ./cmd/fetchcatalog -dir data/catalog  # custom output dir
package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// All sets from Alpha through Onslaught block.
var allSets = []string{
	// Core sets
	"LEA", "LEB", "2ED", "3ED", "4ED", "5ED", "6ED", "7ED",
	// Expansions
	"ARN", "ATQ", "LEG", "DRK", "FEM", "HML",
	"ICE", "ALL", "MIR", "VIS", "WTH",
	"TMP", "STH", "EXO",
	"USG", "ULG", "UDS",
	"MMQ", "NEM", "PCY",
	"INV", "PLS", "APC",
	"ODY", "TOR", "JUD",
	"ONS", "LGN", "SCG",
}

// CardEntry mirrors catalog.CardEntry for JSON output.
type CardEntry struct {
	Name            string            `json:"name"`
	ManaCost        string            `json:"mana_cost,omitempty"`
	CMC             float64           `json:"cmc"`
	TypeLine        string            `json:"type_line"`
	OracleText      string            `json:"oracle_text,omitempty"`
	Power           string            `json:"power,omitempty"`
	Toughness       string            `json:"toughness,omitempty"`
	Colors          []string          `json:"colors"`
	ColorIdentity   []string          `json:"color_identity"`
	Keywords        []string          `json:"keywords"`
	Rarity          string            `json:"rarity"`
	Set             string            `json:"set"`
	SetName         string            `json:"set_name"`
	CollectorNumber string            `json:"collector_number"`
	Artist          string            `json:"artist"`
	FlavorText      string            `json:"flavor_text,omitempty"`
	Layout          string            `json:"layout"`
	Reserved        bool              `json:"reserved"`
	Reprint         bool              `json:"reprint"`
	MultiverseIDs   []int             `json:"multiverse_ids"`
	OracleID        string            `json:"oracle_id"`
	ReleasedAt      string            `json:"released_at"`
	Legalities      map[string]string `json:"legalities,omitempty"`
	CardFaces       []CardFace        `json:"card_faces,omitempty"`
}

type CardFace struct {
	Name       string   `json:"name"`
	ManaCost   string   `json:"mana_cost,omitempty"`
	TypeLine   string   `json:"type_line,omitempty"`
	OracleText string   `json:"oracle_text,omitempty"`
	Power      string   `json:"power,omitempty"`
	Toughness  string   `json:"toughness,omitempty"`
	Artist     string   `json:"artist,omitempty"`
	FlavorText string   `json:"flavor_text,omitempty"`
	Colors     []string `json:"colors,omitempty"`
}

// SetInfo mirrors catalog.SetInfo for JSON output.
type SetInfo struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	SetType    string `json:"set_type"`
	ReleasedAt string `json:"released_at"`
	CardCount  int    `json:"card_count"`
	Block      string `json:"block"`
	BlockCode  string `json:"block_code"`
}

// scryfallCard is the raw Scryfall card object — we extract what we need.
type scryfallCard struct {
	Name            string            `json:"name"`
	ManaCost        string            `json:"mana_cost"`
	CMC             float64           `json:"cmc"`
	TypeLine        string            `json:"type_line"`
	OracleText      string            `json:"oracle_text"`
	Power           string            `json:"power"`
	Toughness       string            `json:"toughness"`
	Colors          []string          `json:"colors"`
	ColorIdentity   []string          `json:"color_identity"`
	Keywords        []string          `json:"keywords"`
	Rarity          string            `json:"rarity"`
	Set             string            `json:"set"`
	SetName         string            `json:"set_name"`
	CollectorNumber string            `json:"collector_number"`
	Artist          string            `json:"artist"`
	FlavorText      string            `json:"flavor_text"`
	Layout          string            `json:"layout"`
	Reserved        bool              `json:"reserved"`
	Reprint         bool              `json:"reprint"`
	MultiverseIDs   []int             `json:"multiverse_ids"`
	OracleID        string            `json:"oracle_id"`
	ReleasedAt      string            `json:"released_at"`
	Legalities      map[string]string `json:"legalities"`
	CardFaces       []scryfallFace    `json:"card_faces"`
}

type scryfallFace struct {
	Name       string   `json:"name"`
	ManaCost   string   `json:"mana_cost"`
	TypeLine   string   `json:"type_line"`
	OracleText string   `json:"oracle_text"`
	Power      string   `json:"power"`
	Toughness  string   `json:"toughness"`
	Artist     string   `json:"artist"`
	FlavorText string   `json:"flavor_text"`
	Colors     []string `json:"colors"`
}

type scryfallSetResponse struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	SetType    string `json:"set_type"`
	ReleasedAt string `json:"released_at"`
	CardCount  int    `json:"card_count"`
	Block      string `json:"block"`
	BlockCode  string `json:"block_code"`
	// error fields
	Status  int    `json:"status"`
	Details string `json:"details"`
}

type scryfallSearchResponse struct {
	Data     []scryfallCard `json:"data"`
	HasMore  bool           `json:"has_more"`
	NextPage string         `json:"next_page"`
	// error fields
	Status  int    `json:"status"`
	Details string `json:"details"`
}

func main() {
	dir := flag.String("dir", "data/catalog", "output directory for catalog JSON files")
	setsFlag := flag.String("sets", "", "comma-separated set codes to fetch (default: all 37 sets)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: fetchcatalog [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Fetches card metadata from Scryfall for sets Alpha through Onslaught block.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	sets := allSets
	if *setsFlag != "" {
		sets = strings.Split(*setsFlag, ",")
		for i := range sets {
			sets[i] = strings.TrimSpace(sets[i])
		}
	}

	// Ensure output directories exist.
	setsDir := filepath.Join(*dir, "_sets")
	if err := os.MkdirAll(setsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating directory: %v\n", err)
		os.Exit(1)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if os.Getenv("FETCHSET_SKIP_TLS") != "" {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // sandbox workaround
	}
	client := &http.Client{Timeout: 30 * time.Second, Transport: transport}

	totalCards := 0
	for i, code := range sets {
		fmt.Fprintf(os.Stderr, "[%d/%d] Fetching %s...\n", i+1, len(sets), code)

		// Fetch set metadata.
		si, err := fetchSetInfo(client, code)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  error fetching set info for %s: %v\n", code, err)
			os.Exit(1)
		}
		setData, _ := json.MarshalIndent(si, "", "  ")
		setFile := filepath.Join(setsDir, strings.ToUpper(code)+".json")
		if err := os.WriteFile(setFile, setData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "  error writing %s: %v\n", setFile, err)
			os.Exit(1)
		}

		// Fetch cards.
		cards, err := fetchCards(client, code)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  error fetching cards for %s: %v\n", code, err)
			os.Exit(1)
		}
		totalCards += len(cards)

		cardData, _ := json.MarshalIndent(cards, "", "  ")
		cardFile := filepath.Join(*dir, strings.ToUpper(code)+".json")
		if err := os.WriteFile(cardFile, cardData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "  error writing %s: %v\n", cardFile, err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "  %s: %d cards\n", si.Name, len(cards))
	}

	fmt.Fprintf(os.Stderr, "\nDone. %d sets, %d total cards in %s\n", len(sets), totalCards, *dir)
}

func fetchSetInfo(client *http.Client, code string) (SetInfo, error) {
	url := fmt.Sprintf("https://api.scryfall.com/sets/%s", strings.ToLower(code))
	var resp scryfallSetResponse
	if err := get(client, url, &resp); err != nil {
		return SetInfo{}, err
	}
	if resp.Status != 0 {
		return SetInfo{}, fmt.Errorf("scryfall: %s (status %d)", resp.Details, resp.Status)
	}
	time.Sleep(80 * time.Millisecond)
	return SetInfo{
		Code:       resp.Code,
		Name:       resp.Name,
		SetType:    resp.SetType,
		ReleasedAt: resp.ReleasedAt,
		CardCount:  resp.CardCount,
		Block:      resp.Block,
		BlockCode:  resp.BlockCode,
	}, nil
}

func fetchCards(client *http.Client, code string) ([]CardEntry, error) {
	url := fmt.Sprintf(
		"https://api.scryfall.com/cards/search?q=set:%s&order=set&unique=prints",
		strings.ToLower(code),
	)

	var all []CardEntry
	for url != "" {
		var page scryfallSearchResponse
		if err := get(client, url, &page); err != nil {
			return nil, err
		}
		if page.Status != 0 {
			return nil, fmt.Errorf("scryfall: %s (status %d)", page.Details, page.Status)
		}

		for _, sc := range page.Data {
			all = append(all, convertCard(sc))
		}

		if page.HasMore {
			url = page.NextPage
			time.Sleep(80 * time.Millisecond)
		} else {
			url = ""
		}
	}
	return all, nil
}

func convertCard(sc scryfallCard) CardEntry {
	entry := CardEntry{
		Name:            sc.Name,
		ManaCost:        sc.ManaCost,
		CMC:             sc.CMC,
		TypeLine:        sc.TypeLine,
		OracleText:      sc.OracleText,
		Power:           sc.Power,
		Toughness:       sc.Toughness,
		Colors:          sc.Colors,
		ColorIdentity:   sc.ColorIdentity,
		Keywords:        sc.Keywords,
		Rarity:          sc.Rarity,
		Set:             sc.Set,
		SetName:         sc.SetName,
		CollectorNumber: sc.CollectorNumber,
		Artist:          sc.Artist,
		FlavorText:      sc.FlavorText,
		Layout:          sc.Layout,
		Reserved:        sc.Reserved,
		Reprint:         sc.Reprint,
		MultiverseIDs:   sc.MultiverseIDs,
		OracleID:        sc.OracleID,
		ReleasedAt:      sc.ReleasedAt,
		Legalities:      sc.Legalities,
	}
	for _, f := range sc.CardFaces {
		entry.CardFaces = append(entry.CardFaces, CardFace(f))
	}
	return entry
}

func get(client *http.Client, url string, dst any) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("User-Agent", "mage-go/fetchcatalog (git.sr.ht/~cdcarter/mage-go)")
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
