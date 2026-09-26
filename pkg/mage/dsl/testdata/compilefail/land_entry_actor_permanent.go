package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.LandEnters().WithActor(dsl.BindPermanent("land"))

// ERROR cannot use
