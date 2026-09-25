package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.RemoveAbilitiesEach(dsl.You)

// ERROR does not implement
