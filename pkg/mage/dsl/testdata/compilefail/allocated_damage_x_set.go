package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DealAllocatedDamageX(dsl.This, dsl.DamageRecipients(dsl.EachPlayer, dsl.MatchingPermanents(dsl.IsCreature)))

// ERROR cannot use dsl.DamageRecipients
