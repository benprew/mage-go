package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DealDamageFrom(dsl.This, dsl.TargetDamageRecipientsAnyNumber("recipients"), dsl.Fixed(1))

// ERROR cannot use dsl.TargetDamageRecipientsAnyNumber
