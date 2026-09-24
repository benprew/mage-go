package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.AtModes(dsl.Enters(dsl.This))

// ERROR does not implement mage.TurnStepPattern
