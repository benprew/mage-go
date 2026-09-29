package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DiscardMatchingCardCostInto(dsl.IsCreature, dsl.BindCard("paid"))
// ERROR cannot use dsl.IsCreature
