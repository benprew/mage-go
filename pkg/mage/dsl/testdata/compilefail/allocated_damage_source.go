package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DealAllocatedDamage(dsl.You, dsl.TargetDamageRecipients("recipients", 1, 3), 3)

// ERROR cannot use dsl.You
