package fixture
import . "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = ActivateModes(Costs(), ActivationManaPerTarget(Red, 1))
// ERROR does not implement mage.ModalActivationOption
