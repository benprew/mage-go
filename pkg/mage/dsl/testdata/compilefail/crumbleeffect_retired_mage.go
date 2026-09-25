package compilefail

import "github.com/benprew/mage-go/pkg/mage"

var _ = mage.CrumbleEffect

// ERROR undefined: mage.CrumbleEffect
