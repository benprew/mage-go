package arabian

import (
	_ "embed"

	"git.sr.ht/~cdcarter/mage-go/pkg/catalog"
)

//go:embed ARN.json
var catalogData []byte

func init() {
	catalog.RegisterSet("ARN", "Arabian Nights", catalogData)
}
