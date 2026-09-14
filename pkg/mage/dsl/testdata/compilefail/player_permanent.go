package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.Sacrifice(dsl.SelectController()) // ERROR PermanentRef
