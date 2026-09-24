package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var amount dsl.ValueSource = dsl.Fixed(2)
var _ = dsl.LoseLifeEach(dsl.EachPlayer, amount)
// ERROR does not implement mage.ResolutionValue
