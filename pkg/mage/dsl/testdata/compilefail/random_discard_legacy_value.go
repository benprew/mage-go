package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.DiscardAtRandom(dsl.You, dsl.XValue())
// ERROR does not implement mage.ResolutionValue
