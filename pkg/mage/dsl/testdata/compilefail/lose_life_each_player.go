package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.LoseLifeEach(dsl.You, dsl.Fixed(3))
// ERROR does not implement mage.PlayerSet
