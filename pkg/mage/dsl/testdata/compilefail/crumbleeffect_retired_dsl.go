package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.CrumbleEffect

// ERROR undefined: dsl.CrumbleEffect
