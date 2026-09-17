package fixture

import . "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = UntilSourceLeaves(You, ModifyPT(This, Fixed(1), Fixed(1)))

// ERROR cannot use You
