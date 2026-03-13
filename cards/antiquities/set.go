package antiquities

import (
	_ "embed"

	"git.sr.ht/~cdcarter/mage-go/pkg/catalog"
)

//go:embed ATQ.json
var catalogData []byte

func init() {
	catalog.RegisterSet("ATQ", "Antiquities", catalogData)
}
