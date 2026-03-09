package arabian

import (
	_ "embed"

	"github.com/mage/mage/pkg/catalog"
)

//go:embed ARN.json
var catalogData []byte

func init() {
	catalog.RegisterSet("ARN", "Arabian Nights", catalogData)
}
