package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.OpponentLosesLife(dsl.BindPlayer("player")).WithAmount(dsl.BindPlayer("amount"))

// ERROR cannot use
