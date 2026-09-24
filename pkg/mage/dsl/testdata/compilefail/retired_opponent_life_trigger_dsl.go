package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.WheneverOpponentLosesLifeTrigger(dsl.GainLife(dsl.You, dsl.Fixed(1)), false)

// ERROR undefined: dsl.WheneverOpponentLosesLifeTrigger
