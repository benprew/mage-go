package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.Spell(dsl.Regenerate(dsl.This), dsl.ActivationIf(dsl.Controls(dsl.IsCreature)))

// ERROR does not implement mage.SpellOption
