package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.Mill(dsl.FromNumber(dsl.BindNumber("amount")), dsl.Fixed(1))
// ERROR does not implement mage.PlayerRef
