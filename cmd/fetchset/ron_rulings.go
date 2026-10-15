package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/benprew/mage-go/pkg/mage/dsl/ron"
)

func ronCardNames(dir string) ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.ron"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no RON files in %s", dir)
	}
	names := make(map[string]bool)
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		nodes, err := ron.Parse(bytes.NewReader(data), path)
		if err != nil {
			return nil, err
		}
		for _, node := range nodes {
			name := node.Fields["Name"].String
			if name == "" {
				return nil, fmt.Errorf("%s: card without a name", path)
			}
			names[name] = true
		}
	}
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	slices.Sort(result)
	return result, nil
}

func fetchRONRulings(client *http.Client, dir string) ([]Card, error) {
	names, err := ronCardNames(dir)
	if err != nil {
		return nil, err
	}
	return fetchCollectionRulings(client, "https://api.scryfall.com", names, 110*time.Millisecond)
}

func fetchCollectionRulings(client *http.Client, baseURL string, names []string, interval time.Duration) ([]Card, error) {
	var cards []Card
	for start := 0; start < len(names); start += 75 {
		identifiers := make([]map[string]string, 0, 75)
		for _, name := range names[start:min(start+75, len(names))] {
			identifiers = append(identifiers, map[string]string{"name": name})
		}
		body, err := json.Marshal(map[string]any{"identifiers": identifiers})
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequest(http.MethodPost, baseURL+"/cards/collection", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		var page struct {
			Data     []Card            `json:"data"`
			NotFound []json.RawMessage `json:"not_found"`
		}
		if err := requestJSON(client, req, &page); err != nil {
			return nil, err
		}
		if len(page.NotFound) != 0 || len(page.Data) != len(identifiers) {
			return nil, fmt.Errorf("incomplete card lookup: requested %d, received %d, not found: %s", len(identifiers), len(page.Data), page.NotFound)
		}
		cards = append(cards, page.Data...)
		fmt.Fprintf(os.Stderr, "resolved %d / %d RON card names\n", len(cards), len(names))
		time.Sleep(interval)
	}
	var bulk struct {
		DownloadURI      string `json:"download_uri"`
		JSONLDownloadURI string `json:"jsonl_download_uri"`
	}
	if err := get(client, baseURL+"/bulk-data/rulings", &bulk); err != nil {
		return nil, err
	}
	if bulk.DownloadURI == "" {
		bulk.DownloadURI = bulk.JSONLDownloadURI
	}
	if bulk.DownloadURI == "" {
		return nil, fmt.Errorf("rulings bulk data has no download URL")
	}
	time.Sleep(interval)
	fmt.Fprintln(os.Stderr, "downloading bulk rulings")
	req, err := http.NewRequest(http.MethodGet, bulk.DownloadURI, http.NoBody)
	if err != nil {
		return nil, err
	}
	data, err := requestBytes(client, req)
	if err != nil {
		return nil, err
	}
	rulings, err := decodeBulkRulings(data)
	if err != nil {
		return nil, err
	}
	if len(rulings) == 0 {
		return nil, fmt.Errorf("rulings bulk data is empty")
	}
	byOracle := make(map[string][]Ruling)
	for _, ruling := range rulings {
		byOracle[ruling.OracleID] = append(byOracle[ruling.OracleID], ruling)
	}
	for i := range cards {
		cards[i].Rulings = append([]Ruling{}, byOracle[cards[i].OracleID]...)
	}
	return cards, nil
}

func decodeBulkRulings(data []byte) ([]Ruling, error) {
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		data, err = io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
	}
	data = bytes.TrimSpace(data)
	var rulings []Ruling
	if len(data) > 0 && data[0] == '[' {
		if err := json.Unmarshal(data, &rulings); err != nil {
			return nil, err
		}
		return rulings, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var ruling Ruling
		if err := decoder.Decode(&ruling); errors.Is(err, io.EOF) {
			return rulings, nil
		} else if err != nil {
			return nil, err
		}
		rulings = append(rulings, ruling)
	}
}
