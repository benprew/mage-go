package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.GrantLandPlays(dsl.This, dsl.Fixed(1))
// ERROR does not implement mage.PlayerRef
