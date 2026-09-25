package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.ReturnCardToHand(dsl.You)

// ERROR does not implement mage.CardRef
