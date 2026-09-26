package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.ManaValueOfSpell(dsl.You)

// ERROR cannot use
