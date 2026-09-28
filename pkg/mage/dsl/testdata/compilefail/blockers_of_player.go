package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.BlockersOf(dsl.You)

// ERROR does not implement mage.PermanentRef
