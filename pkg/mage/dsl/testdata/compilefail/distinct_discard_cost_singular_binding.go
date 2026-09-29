package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DiscardCardsWithDifferentNamesCostWithCards(2, dsl.BindCard("paid"))

// ERROR cannot use
