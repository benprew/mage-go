package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.TapAttachedCreature

// ERROR undefined: dsl.TapAttachedCreature
