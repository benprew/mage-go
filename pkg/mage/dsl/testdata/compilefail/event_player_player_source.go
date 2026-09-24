package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DealsDamageToPlayer(dsl.You, dsl.BindPlayer("player"))

// ERROR cannot use
