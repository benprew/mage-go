package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.SacrificeCreatureCostWithStats(dsl.BindPermanent("creature"), dsl.BindNumber("toughness"))

// ERROR cannot use
