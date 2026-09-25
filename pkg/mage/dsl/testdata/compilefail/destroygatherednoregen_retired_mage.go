package compilefail

import "github.com/benprew/mage-go/pkg/mage"

var _ = mage.DestroyGatheredNoRegen

// ERROR undefined: mage.DestroyGatheredNoRegen
