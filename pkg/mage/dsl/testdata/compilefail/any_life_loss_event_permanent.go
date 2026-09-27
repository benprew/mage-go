package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.PlayerLosesLife(dsl.BindPermanent("loser")) // ERROR *mage.PlayerBinding
