package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.AtNext(dsl.Controls(dsl.IsCreature), dsl.GainLife(dsl.You, dsl.Fixed(1)))
// ERROR does not implement mage.DelayedStepPattern
