package compilefail

import "github.com/benprew/mage-go/pkg/mage"

var _ = mage.DestroyAllMatching

// ERROR undefined: mage.DestroyAllMatching
