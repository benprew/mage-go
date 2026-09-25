package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.ControllerGainsLife(dsl.BindPermanent("gainer")) // ERROR *mage.PlayerBinding
