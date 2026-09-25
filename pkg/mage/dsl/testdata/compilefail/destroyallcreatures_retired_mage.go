package compilefail

import "github.com/benprew/mage-go/pkg/mage"

var _ = mage.DestroyAllCreatures

// ERROR undefined: mage.DestroyAllCreatures
