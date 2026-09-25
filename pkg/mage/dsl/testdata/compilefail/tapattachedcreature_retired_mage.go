package compilefail

import "github.com/benprew/mage-go/pkg/mage"

var _ = mage.TapAttachedCreature

// ERROR undefined: mage.TapAttachedCreature
