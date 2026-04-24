package fourthedition

import (
	_ "embed"

	"git.sr.ht/~cdcarter/mage-go/pkg/catalog"
)

//go:embed 4ED.json
var catalogData []byte

func init() {
	catalog.RegisterSet("4ED", "Fourth Edition", catalogData)
}
