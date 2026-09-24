package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var legacy dsl.ValueSource = dsl.Fixed(1)
var _ = dsl.SubtractValues(legacy, dsl.Fixed(1))

// ERROR does not implement mage.ResolutionValue
