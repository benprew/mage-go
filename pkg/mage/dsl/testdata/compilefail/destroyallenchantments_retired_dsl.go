package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DestroyAllEnchantments

// ERROR undefined: dsl.DestroyAllEnchantments
