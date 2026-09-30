package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DealAllocatedDamageX(dsl.You, dsl.TargetDamageRecipientsUpToX("recipients"))

// ERROR cannot use dsl.You
