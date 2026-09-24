package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.GrantEachActivatedAbility(dsl.MatchingPermanents(dsl.IsCreature), dsl.Costs(), dsl.Regenerate(dsl.This), dsl.Controls(dsl.IsCreature))

// ERROR does not implement mage.GrantedActivationOption
