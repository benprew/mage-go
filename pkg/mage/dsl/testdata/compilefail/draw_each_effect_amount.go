package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DrawEach(dsl.EachPlayer, dsl.Destroy(dsl.This))

// ERROR does not implement mage.ResolutionValue
