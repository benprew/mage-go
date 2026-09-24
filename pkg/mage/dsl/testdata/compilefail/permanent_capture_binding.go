package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.CapturePermanentBeforePayment(dsl.This, dsl.BindNumber("host")) // ERROR PermanentBinding
