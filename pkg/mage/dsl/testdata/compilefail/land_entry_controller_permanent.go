package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.LandEnters().WithController(dsl.BindPermanent("land"))

// ERROR cannot use
