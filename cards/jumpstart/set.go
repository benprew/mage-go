package jumpstart

import (
	_ "embed"

	"git.sr.ht/~cdcarter/mage-go/pkg/catalog"
)

//go:embed JMP.json
var catalogData []byte

func init() {
	catalog.RegisterSet("JMP", "Jumpstart", catalogData)
}
