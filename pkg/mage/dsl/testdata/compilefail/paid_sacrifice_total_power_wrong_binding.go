package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.SacrificeNMatchingCostWithTotalPower(2, dsl.IsCreature, "Sacrifice two", dsl.BindCard("total")) // ERROR NumberBinding
