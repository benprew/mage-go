package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.WhenEvent(dsl.YourFirstMainPhase, dsl.Draw(dsl.You, dsl.Fixed(1)))

// ERROR does not implement mage.EventPattern
