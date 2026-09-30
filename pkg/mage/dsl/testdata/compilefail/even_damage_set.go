package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DealDamageEvenly(dsl.This, dsl.DamageRecipients(dsl.EachPlayer, dsl.MatchingPermanents(dsl.IsCreature)), dsl.Fixed(5))

// ERROR cannot use dsl.DamageRecipients
