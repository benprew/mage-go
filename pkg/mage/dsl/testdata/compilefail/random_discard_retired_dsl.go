package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DiscardRandom(1)

// ERROR undefined: dsl.DiscardRandom
