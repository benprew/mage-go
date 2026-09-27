package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.TapEach(dsl.This) // ERROR does not implement mage.PermanentSet
