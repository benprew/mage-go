package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.ActivateModes(dsl.Costs(), dsl.GainLife(dsl.You, dsl.Fixed(1)))

// ERROR cannot use
