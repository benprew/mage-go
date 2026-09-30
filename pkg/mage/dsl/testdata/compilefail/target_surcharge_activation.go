package fixture
import . "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = Activate(Costs(), Draw(You, Fixed(1)), AdditionalManaPerTargetBeyondFirst(1))
// ERROR does not implement mage.ActivationOption
