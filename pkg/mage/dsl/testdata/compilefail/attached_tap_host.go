package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.AttachedBecomesTapped(dsl.BindPlayer("host")) // ERROR *mage.PermanentBinding
