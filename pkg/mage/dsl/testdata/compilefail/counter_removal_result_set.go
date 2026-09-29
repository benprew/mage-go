package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.RemoveCountersFromWithResult(dsl.ControlledPermanents(dsl.IsCreature), dsl.Charge, dsl.Fixed(1), dsl.BindNumber("removed"))
// ERROR does not implement mage.PermanentRef
