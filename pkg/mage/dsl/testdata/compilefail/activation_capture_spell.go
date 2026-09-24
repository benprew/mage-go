package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.Spell(dsl.Draw(dsl.You, dsl.Fixed(1)), dsl.CaptureNumberBeforePayment(dsl.Fixed(1), dsl.BindNumber("n"))) // ERROR SpellOption
