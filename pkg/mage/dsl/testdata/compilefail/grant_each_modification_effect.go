package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.Sequence(dsl.EachActivatedAbilityModification(dsl.MatchingPermanents(dsl.IsCreature), dsl.Costs(), dsl.Regenerate(dsl.This)))

// ERROR ResolvingEffect
