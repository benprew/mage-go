package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.PlayersDealtDamageBy(dsl.This, dsl.MatchingPermanents(dsl.IsCreature))

// ERROR mage.PermanentSet does not implement mage.PlayerSet
