package compilefail

import "github.com/benprew/mage-go/pkg/mage"

var _ = mage.SwordsToPlowsharesEffect()

// ERROR undefined: mage.SwordsToPlowsharesEffect
