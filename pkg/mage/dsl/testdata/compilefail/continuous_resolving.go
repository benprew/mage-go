package fixture
import . "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = Sequence(BoostAttached(1,1,AttachAura))
// ERROR does not implement mage.ResolvingEffect
