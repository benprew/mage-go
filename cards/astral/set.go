package astral

import (
	_ "embed"

	"github.com/benprew/mage-go/pkg/catalog"
)

//go:embed PAST.json
var catalogData []byte

func init() {
	catalog.RegisterSet("PAST", "Astral Cards", catalogData)
}
