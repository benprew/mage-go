package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.GainLife(dsl.This, dsl.Fixed(2))
// ERROR does not implement mage.PlayerRef
