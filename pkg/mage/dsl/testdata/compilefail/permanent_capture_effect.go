package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.Sequence(dsl.CapturePermanentBeforePayment(dsl.This, dsl.BindPermanent("host"))) // ERROR ResolvingEffect
