package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.ChooseRandomPlayer(dsl.MatchingPermanents(dsl.IsCreature), dsl.BindPlayer("player"))
// ERROR does not implement mage.PlayerSet
