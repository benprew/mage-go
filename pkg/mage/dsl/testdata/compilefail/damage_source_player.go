package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DamageSourceOf(dsl.You)

// ERROR cannot use dsl.You
