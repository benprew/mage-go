package antiquities

import (
	_ "embed"

	"github.com/mage/mage/pkg/catalog"
)

//go:embed ATQ.json
var catalogData []byte

func init() {
	catalog.RegisterSet("ATQ", "Antiquities", catalogData)
}
