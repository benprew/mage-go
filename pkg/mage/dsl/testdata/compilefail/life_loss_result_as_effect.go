package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.Sequence(dsl.LifeLostInto(dsl.BindNumber("life lost")))

// ERROR does not implement mage.ResolvingEffect
