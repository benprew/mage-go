package fixture

import . "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = Fight(MatchingPermanents(IsCreature), This)

// ERROR does not implement mage.PermanentRef
