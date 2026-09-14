package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.Sequence(dsl.Controls(dsl.IsCreature)) // ERROR ResolvingEffect
