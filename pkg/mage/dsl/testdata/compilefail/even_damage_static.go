package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.Static(dsl.DealDamageEvenly(dsl.This, dsl.TargetDamageRecipientsAnyNumber("recipients"), dsl.Fixed(5)))

// ERROR cannot use dsl.DealDamageEvenly
