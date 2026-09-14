package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.Snapshot(dsl.Controls(dsl.IsCreature), dsl.BindPermanent("host"))
// ERROR does not implement mage.PermanentRef
