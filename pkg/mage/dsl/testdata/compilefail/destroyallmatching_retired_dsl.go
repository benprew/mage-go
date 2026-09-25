package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DestroyAllMatching

// ERROR undefined: dsl.DestroyAllMatching
