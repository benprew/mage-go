package fixture

import "github.com/benprew/mage-go/pkg/mage"

var _ = mage.WheneverOpponentLosesLifeTrigger(mage.GainLifeFor(mage.You, mage.Fixed(1)), false)

// ERROR undefined: mage.WheneverOpponentLosesLifeTrigger
