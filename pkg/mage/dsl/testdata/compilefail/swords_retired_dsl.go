package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.SwordsToPlowsharesEffect()

// ERROR undefined: dsl.SwordsToPlowsharesEffect
