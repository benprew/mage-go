package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.CapturePermanentBeforePayment(dsl.You, dsl.BindPermanent("host")) // ERROR PermanentRef
