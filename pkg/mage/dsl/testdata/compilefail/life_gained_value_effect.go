package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.Sequence(dsl.LifeGainedThisTurn(dsl.You))

// ERROR does not implement mage.ResolvingEffect
