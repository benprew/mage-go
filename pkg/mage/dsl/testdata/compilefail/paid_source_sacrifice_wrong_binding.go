package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.SacrificeCreatureIncludingSourceCostWithStats(dsl.BindPermanent("creature"), dsl.BindNumber("toughness")) // ERROR NumberBinding
