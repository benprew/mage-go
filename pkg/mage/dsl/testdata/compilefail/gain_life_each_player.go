package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.GainLifeEach(dsl.You, dsl.Fixed(1))

// ERROR does not implement mage.PlayerSet
