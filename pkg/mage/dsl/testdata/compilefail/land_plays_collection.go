package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.GrantLandPlays(dsl.EachPlayer, dsl.Fixed(1))
// ERROR does not implement mage.PlayerRef
