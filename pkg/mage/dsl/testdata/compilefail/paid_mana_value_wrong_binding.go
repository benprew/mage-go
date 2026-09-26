package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.SacrificeMatchingCostWithManaValue(dsl.IsArtifact, "Sacrifice an artifact", dsl.BindPermanent("paid")) // ERROR NumberBinding
