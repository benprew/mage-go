package fixture

import . "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = AtNext(EachEndStep, Draw(You, Fixed(1)), WithInterveningIf(GainLife(You, Fixed(1))))

// ERROR does not implement mage.StateCondition
