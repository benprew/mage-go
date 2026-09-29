package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DiscardCostWithCards(2, dsl.BindCard("paid"))
// ERROR cannot use
