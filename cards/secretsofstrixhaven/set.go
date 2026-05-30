package secretsofstrixhaven

import (
	_ "embed"

	"github.com/benprew/mage-go/pkg/catalog"
)

//go:embed SOS.json
var catalogData []byte

func init() {
	catalog.RegisterSet("SOS", "Secrets of Strixhaven", catalogData)
}
