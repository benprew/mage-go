package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.SkipNextUntapOf(dsl.You, dsl.You)
// ERROR does not implement mage.PermanentRef
