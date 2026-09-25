package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DestroyAllCreatures

// ERROR undefined: dsl.DestroyAllCreatures
