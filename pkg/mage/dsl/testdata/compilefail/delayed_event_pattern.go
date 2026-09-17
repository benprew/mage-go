package fixture

import . "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = AtNext(Dies(This), Destroy(This))

// ERROR does not implement mage.TurnStepPattern
