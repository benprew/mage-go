package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DealDamageFrom(dsl.This, dsl.TargetDamageRecipients("targets", 1, 2), dsl.Fixed(1))

// ERROR does not implement mage.DamageRecipientRef
