package legends

import (
	_ "embed"

	"github.com/mage/mage/pkg/catalog"
)

//go:embed LEG.json
var catalogData []byte

func init() {
	catalog.RegisterSet("LEG", "Legends", catalogData)
}
