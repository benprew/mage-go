package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.WhenEventModes(dsl.YourUpkeep)

// ERROR does not implement mage.EventPattern
