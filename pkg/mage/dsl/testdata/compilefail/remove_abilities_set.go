package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.RemoveAbilities(dsl.MatchingPermanents(dsl.IsCreature))

// ERROR does not implement
