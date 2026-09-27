package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.SacrificeCreatureCostWithSubtype("Thrull", dsl.BindPermanent("paid")) // ERROR NumberBinding
