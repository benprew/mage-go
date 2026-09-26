package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.BecomesTargetOfSpell(dsl.You) // ERROR sourceRef
