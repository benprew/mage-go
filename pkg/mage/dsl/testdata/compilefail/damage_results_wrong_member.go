package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.ForEachDamageResult(dsl.BindDamageResults("results"), dsl.BindNumber("number"), dsl.GainLife(dsl.You, dsl.Fixed(1)))

// ERROR cannot use
