package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.AttachedControllerUpkeep(dsl.BindPlayer("host"), dsl.BindPlayer("player")) // ERROR *mage.PermanentBinding
