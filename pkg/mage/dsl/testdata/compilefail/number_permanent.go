package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.FromNumber(dsl.BindPermanent("amount"))
// ERROR cannot use
