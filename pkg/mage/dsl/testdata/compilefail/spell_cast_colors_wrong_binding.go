package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.SpellCast().WithColorsSpent(dsl.BindPlayer("player"))

// ERROR cannot use
