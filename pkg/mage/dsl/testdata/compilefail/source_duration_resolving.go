package fixture

import . "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = UntilSourceLeaves(This, Draw(You, Fixed(1)))

// ERROR does not implement mage.ContinuousModification
