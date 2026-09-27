package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.SacrificeMatchingIncludingSourceCostWithManaValue(dsl.IsCreature, "Sacrifice a creature", dsl.BindPermanent("paid")) // ERROR NumberBinding
