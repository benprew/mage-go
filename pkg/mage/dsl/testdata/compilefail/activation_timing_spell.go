package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.Spell(dsl.Regenerate(dsl.This), dsl.OnlyAsSorcery())

// ERROR does not implement mage.SpellOption
