package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.MayTapOrUntap(dsl.MatchingPermanents(dsl.IsCreature))
// ERROR does not implement mage.PermanentRef
