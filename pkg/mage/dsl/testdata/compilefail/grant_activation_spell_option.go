package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.GrantEachActivatedAbility(dsl.MatchingPermanents(dsl.IsCreature), dsl.Costs(), dsl.Regenerate(dsl.This), dsl.WithAdditionalCost(dsl.Mana("{1}")))

// ERROR does not implement mage.GrantedActivationOption
