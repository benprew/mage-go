package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.ControllerDrawsCard(dsl.BindPermanent("drawer")) // ERROR *mage.PlayerBinding
