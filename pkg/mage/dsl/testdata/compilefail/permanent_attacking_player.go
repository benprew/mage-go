package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.PermanentIsAttacking(dsl.You) // ERROR cannot use
