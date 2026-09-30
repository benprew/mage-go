package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DealDamageEvenly(dsl.You, dsl.TargetDamageRecipientsAnyNumber("recipients"), dsl.Fixed(5))

// ERROR cannot use dsl.You
