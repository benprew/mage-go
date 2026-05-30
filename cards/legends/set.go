package legends

import (
	_ "embed"

	"github.com/benprew/mage-go/pkg/catalog"
)

//go:embed LEG.json
var catalogData []byte

func init() {
	catalog.RegisterSet("LEG", "Legends", catalogData)
}
