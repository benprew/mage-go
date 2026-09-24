package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.SnapshotPlayer(dsl.EachPlayer, dsl.BindPlayer("player")) // ERROR does not implement mage.PlayerRef
