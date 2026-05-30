package legends

import . "github.com/benprew/mage-go/pkg/mage/dsl"

// basicLandNames are the five basic land card names.
var basicLandNames = map[string]bool{
	"Plains": true, "Island": true, "Swamp": true, "Mountain": true, "Forest": true,
}

// isBasicLand returns true if a card is a basic land (by name).
func isBasicLand(c Card) bool {
	return basicLandNames[c.Name()]
}
