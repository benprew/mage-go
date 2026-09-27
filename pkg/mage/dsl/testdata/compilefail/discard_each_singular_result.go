package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DiscardEach(dsl.EachPlayer, dsl.DiscardAmountInto(dsl.BindNumber("count")))

// ERROR does not implement mage.DiscardEachResultOption
