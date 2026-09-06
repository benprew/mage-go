package thedark

import (
	_ "embed"

	"github.com/benprew/mage-go/pkg/catalog"
)

//go:embed DRK.json
var catalogData []byte

func init() {
	catalog.RegisterSet("DRK", "The Dark", catalogData)
}
