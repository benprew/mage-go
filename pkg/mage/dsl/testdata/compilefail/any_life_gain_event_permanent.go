package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.PlayerGainsLife(dsl.BindPermanent("gainer")) // ERROR *mage.PlayerBinding
