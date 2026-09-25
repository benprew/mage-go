package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DestroyGatheredNoRegen

// ERROR undefined: dsl.DestroyGatheredNoRegen
