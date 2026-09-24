package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DealsDamageToPlayer(dsl.This, dsl.BindPermanent("player"))

// ERROR cannot use
