package fixture

import . "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = Static(UntilSourceLeaves(This, ModifyPT(This, Fixed(1), Fixed(1))))

// ERROR does not implement mage.StaticRule
