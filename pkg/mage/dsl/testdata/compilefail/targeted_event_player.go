package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.BecomesTargeted(dsl.You) // ERROR sourceRef
