package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.GainLifeEach(dsl.PlayerWithUniqueHighestLife(), dsl.Fixed(1)) // ERROR PlayerSet
