package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.AddCountersEach(dsl.This, dsl.P1P1, dsl.Fixed(1)) // ERROR does not implement mage.PermanentSet
