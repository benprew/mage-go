package compilefail

import dsl "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.BecomesTapped(dsl.You) // ERROR cannot use
