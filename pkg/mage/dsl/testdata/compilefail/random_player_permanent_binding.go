package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.ChooseRandomPlayer(dsl.EachPlayer, dsl.BindPermanent("permanent"))
// ERROR cannot use
