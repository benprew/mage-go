package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.UntapPermanentWithResult(dsl.You, dsl.UntappedThisWayInto(dsl.BindNumber("result")))
// ERROR does not implement mage.PermanentRef
