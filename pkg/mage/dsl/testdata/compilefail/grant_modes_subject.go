package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.GrantEachActivatedModes(dsl.This, dsl.Costs(), dsl.ModeOf("gain", dsl.GainLife(dsl.You, dsl.Fixed(1))), dsl.ModeOf("regenerate", dsl.Regenerate(dsl.This)))

// ERROR does not implement mage.PermanentSet
