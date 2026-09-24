package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.AtNext(dsl.ControllerDrawsCard(dsl.BindPlayer("drawer")), dsl.Draw(dsl.You, dsl.Fixed(1))) // ERROR missing method delayedStep
