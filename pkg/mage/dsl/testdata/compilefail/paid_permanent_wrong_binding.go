package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.SacrificeMatchingCostWithPermanent(dsl.IsCreature, "Sacrifice a creature", dsl.BindNumber("paid")) // ERROR PermanentBinding
