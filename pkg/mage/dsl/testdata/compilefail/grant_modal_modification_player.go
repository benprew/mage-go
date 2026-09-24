package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.ActivatedModesModification(dsl.You, dsl.Costs())

// ERROR PermanentRef
