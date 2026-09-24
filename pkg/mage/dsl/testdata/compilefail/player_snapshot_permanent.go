package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.SnapshotPlayer(dsl.This, dsl.BindPlayer("player")) // ERROR does not implement mage.PlayerRef
