package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.OpponentsOf(dsl.This)
// ERROR does not implement mage.PlayerRef
