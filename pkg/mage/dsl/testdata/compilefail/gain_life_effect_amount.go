package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.GainLife(dsl.You, dsl.Draw(dsl.You, dsl.Fixed(1)))
// ERROR does not implement mage.ResolutionValue
