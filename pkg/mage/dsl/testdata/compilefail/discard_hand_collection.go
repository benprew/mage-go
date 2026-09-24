package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.DiscardEntireHand(dsl.EachPlayer)
// ERROR does not implement mage.PlayerRef
