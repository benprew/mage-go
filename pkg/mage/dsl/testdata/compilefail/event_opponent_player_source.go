package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DealsDamageToOpponent(dsl.You, dsl.BindPlayer("opponent"))

// ERROR cannot use
