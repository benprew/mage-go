package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.Sequence(dsl.SkipDrawStepForLife(2))
// ERROR does not implement mage.ResolvingEffect
