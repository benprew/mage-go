package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.TargetPlayer("player", dsl.TargetYouControl())
// ERROR does not implement mage.PlayerTargetOption
