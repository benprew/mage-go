package promo

import (
	_ "embed"

	"github.com/benprew/mage-go/pkg/catalog"
)

//go:embed PHPR.json
var catalogData []byte

func init() {
	catalog.RegisterSet("PHPR", "HarperPrism Book Promos", catalogData)
}
