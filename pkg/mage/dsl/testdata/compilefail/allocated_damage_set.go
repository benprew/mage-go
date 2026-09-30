package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DealAllocatedDamage(dsl.This, dsl.DamageRecipients(dsl.EachPlayer, dsl.MatchingPermanents(dsl.IsCreature)), 3)

// ERROR cannot use dsl.DamageRecipients
