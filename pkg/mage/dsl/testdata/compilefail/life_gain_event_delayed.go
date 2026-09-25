package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.AtNext(dsl.ControllerGainsLife(dsl.BindPlayer("gainer")), dsl.Draw(dsl.You, dsl.Fixed(1))) // ERROR missing method delayedStep
