package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.Destroy(dsl.FromPlayer(dsl.BindPlayer("player")))
// ERROR does not implement mage.PermanentRef
