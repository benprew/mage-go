package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.BlockedAttackersOf(dsl.You)

// ERROR does not implement mage.PermanentRef
