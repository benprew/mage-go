package antiquities

import (
	_ "embed"

	"github.com/benprew/mage-go/pkg/catalog"
)

//go:embed ATQ.json
var catalogData []byte

func init() {
	catalog.RegisterSet("ATQ", "Antiquities", catalogData)
}
