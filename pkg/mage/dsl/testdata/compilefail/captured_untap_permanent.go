package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.SkipNextUntapOf(dsl.This, dsl.This)
// ERROR does not implement mage.PlayerRef
