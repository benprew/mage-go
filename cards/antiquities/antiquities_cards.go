package antiquities

import (
	"path/filepath"
	"runtime"

	"github.com/mage/mage/pkg/catalog"
)

// antiquitiesCards contains every card name originally printed in the Antiquities expansion.
// Used by Golgothian Sylex to identify permanents to sacrifice.
// Built from pkg/catalog data at init time.
var antiquitiesCards map[string]bool

func init() {
	antiquitiesCards = loadAntiquitiesNames()
}

func loadAntiquitiesNames() map[string]bool {
	// Locate the catalog JSON relative to this source file.
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	catalogPath := filepath.Join(root, "data", "catalog", "ATQ.json")

	cat, err := catalog.LoadSet(catalogPath)
	if err != nil {
		panic("failed to load Antiquities catalog: " + err.Error())
	}
	cards := cat.CardsBySet("ATQ")
	names := make(map[string]bool, len(cards))
	seen := make(map[string]bool)
	for _, c := range cards {
		if !seen[c.Name] {
			names[c.Name] = true
			seen[c.Name] = true
		}
	}
	return names
}
