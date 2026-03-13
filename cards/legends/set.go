package legends

import (
	_ "embed"

	"git.sr.ht/~cdcarter/mage-go/pkg/catalog"
)

//go:embed LEG.json
var catalogData []byte

func init() {
	catalog.RegisterSet("LEG", "Legends", catalogData)
}
