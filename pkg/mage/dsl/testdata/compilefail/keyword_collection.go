package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.GrantKeyword(dsl.MatchingPermanents(dsl.IsCreature), dsl.Flying)

// ERROR does not implement mage.PermanentRef
