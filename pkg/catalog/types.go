// Package catalog provides a Scryfall-sourced card metadata catalog
// covering sets from Alpha through Onslaught block. It is independent
// of the game engine (pkg/mage) and correlates by card name.
package catalog

// SetInfo holds set-level metadata from Scryfall.
type SetInfo struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	SetType    string `json:"set_type"`
	ReleasedAt string `json:"released_at"`
	CardCount  int    `json:"card_count"`
	Block      string `json:"block"`
	BlockCode  string `json:"block_code"`
}

// CardEntry holds card-level metadata for a single printing.
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

// CardFace holds per-face data for split/double-faced cards.
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
