package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.TurnStepBegins(dsl.EachUpkeep, dsl.BindPermanent("land")) // ERROR *mage.PlayerBinding
