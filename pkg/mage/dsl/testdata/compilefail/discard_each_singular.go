package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DiscardEach(dsl.You)

// ERROR does not implement mage.PlayerSet
