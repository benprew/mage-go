package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.ControllerOfSpell(dsl.You)

// ERROR cannot use
