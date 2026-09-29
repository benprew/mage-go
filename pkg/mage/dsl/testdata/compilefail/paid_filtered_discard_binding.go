package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DiscardMatchingCardCostInto(dsl.IsArtifactCard, dsl.BindCards("paid"))
// ERROR cannot use dsl.BindCards
