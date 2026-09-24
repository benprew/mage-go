package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ dsl.ValueSource = dsl.FromNumber(dsl.BindNumber("amount"))
// ERROR does not implement mage.ValueSource
