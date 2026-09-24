package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.DealsDamage(dsl.You, dsl.BindNumber("damage"))
// ERROR cannot use
