package jumpstart

import (
	_ "embed"

	"github.com/benprew/mage-go/pkg/catalog"
)

//go:embed JMP.json
var catalogData []byte

func init() {
	catalog.RegisterSet("JMP", "Jumpstart", catalogData)
}
