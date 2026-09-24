package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.EachActivatedAbilityModification(dsl.This, dsl.Costs(), dsl.Regenerate(dsl.This))

// ERROR PermanentSet
