package fixture

import . "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = Destroy(MatchingPermanents(IsCreature))

// ERROR does not implement mage.PermanentRef
