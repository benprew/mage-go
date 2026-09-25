package compilefail

import "github.com/benprew/mage-go/pkg/mage"

var _ = mage.DestroyAllEnchantments

// ERROR undefined: mage.DestroyAllEnchantments
