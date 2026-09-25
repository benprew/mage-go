package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DestroyGathered

// ERROR undefined: dsl.DestroyGathered
