package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.TargetPermanent("permanent", dsl.AnyPermanent, dsl.TargetOpponentOnly())
// ERROR does not implement mage.PermanentTargetOption
