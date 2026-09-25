package compilefail

import "github.com/benprew/mage-go/pkg/mage"

var _ = mage.DestroyGathered

// ERROR undefined: mage.DestroyGathered
