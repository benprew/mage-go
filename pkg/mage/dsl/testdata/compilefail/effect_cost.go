package fixture

import . "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = Costs(Draw(You, Fixed(1)))

// ERROR does not implement mage.Cost
