package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.GrantEachActivatedModes(dsl.MatchingPermanents(dsl.IsCreature), dsl.Costs(), dsl.Regenerate(dsl.This))

// ERROR cannot use
