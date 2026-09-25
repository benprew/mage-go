package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var legacy dsl.ValueSource = dsl.Fixed(2)
var _ = dsl.MultiplyValues(dsl.Fixed(2), legacy)

// ERROR does not implement mage.ResolutionValue
