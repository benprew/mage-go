package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.Spell(dsl.Regenerate(dsl.This), dsl.CapturePermanentBeforePayment(dsl.This, dsl.BindPermanent("host"))) // ERROR SpellOption
