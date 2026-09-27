package fixture

import . "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = WhenDiesThisTurn(You, GainLife(You, Fixed(1)))

// ERROR does not implement mage.PermanentRef
