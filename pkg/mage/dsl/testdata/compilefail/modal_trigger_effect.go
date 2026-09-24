package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.WhenEventModes(dsl.Enters(dsl.This), dsl.Draw(dsl.You, dsl.Fixed(1)))

// ERROR as mage.ModeDefinition value
