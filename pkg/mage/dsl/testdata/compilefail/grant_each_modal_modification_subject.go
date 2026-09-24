package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.EachActivatedModesModification(dsl.This, dsl.Costs())

// ERROR PermanentSet
