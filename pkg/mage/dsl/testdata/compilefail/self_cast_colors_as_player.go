package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.GainLife(dsl.ColorsSpentToCastThisSpell(), dsl.Fixed(1))

// ERROR cannot use
