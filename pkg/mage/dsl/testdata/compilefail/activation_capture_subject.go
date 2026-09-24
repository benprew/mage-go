package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.CaptureNumberBeforePayment(dsl.This, dsl.BindNumber("n")) // ERROR ResolutionValue
