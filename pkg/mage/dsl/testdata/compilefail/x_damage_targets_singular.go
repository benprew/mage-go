package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DealDamageFrom(dsl.This, dsl.TargetDamageRecipientsUpToX("recipients"), dsl.Fixed(1))

// ERROR does not implement mage.DamageRecipientRef
