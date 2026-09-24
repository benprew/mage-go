package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DealsDamageToOpponent(dsl.This, dsl.BindPermanent("opponent"))

// ERROR cannot use
