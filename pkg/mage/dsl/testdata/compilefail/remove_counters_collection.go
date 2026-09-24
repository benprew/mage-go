package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.RemoveCountersFrom(dsl.MatchingPermanents(dsl.IsCreature), dsl.Charge, dsl.Fixed(1))
// ERROR does not implement mage.PermanentRef
