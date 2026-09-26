package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.AttacksUnblocked(dsl.You) // ERROR sourceRef
