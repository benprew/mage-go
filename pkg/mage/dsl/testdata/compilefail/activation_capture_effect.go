package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.Sequence(dsl.CaptureNumberBeforePayment(dsl.Fixed(1), dsl.BindNumber("n"))) // ERROR ResolvingEffect
