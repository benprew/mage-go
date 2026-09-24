package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.AtNext(dsl.AttachedBecomesTapped(dsl.BindPermanent("host")), dsl.Draw(dsl.You, dsl.Fixed(1))) // ERROR missing method delayedStep
