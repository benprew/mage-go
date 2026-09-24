package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.WithoutPermanent(dsl.MatchingPermanents(dsl.IsCreature), dsl.You)
// ERROR does not implement mage.PermanentRef
