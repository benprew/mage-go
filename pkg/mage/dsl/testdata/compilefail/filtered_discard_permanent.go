package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DiscardMatchingCardCost(dsl.IsArtifact)

// ERROR cannot use dsl.IsArtifact
