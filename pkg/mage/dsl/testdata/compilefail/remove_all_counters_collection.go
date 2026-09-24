package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.RemoveAllCounters(dsl.MatchingPermanents(dsl.IsCreature), dsl.Tide)
// ERROR does not implement mage.PermanentRef
