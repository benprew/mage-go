package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.Sacrifice(dsl.BindPermanent("host"))
// ERROR does not implement mage.PermanentRef
