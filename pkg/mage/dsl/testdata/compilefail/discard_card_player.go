package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DiscardCard(dsl.You) // ERROR CardRef
