package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.ExileMatchingCardFromGraveyardCostWithCard(dsl.IsCreatureCard, "Exile", dsl.BindNumber("paid")) // ERROR CardBinding
