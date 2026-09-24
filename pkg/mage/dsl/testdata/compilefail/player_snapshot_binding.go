package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.SnapshotPlayer(dsl.You, dsl.BindPermanent("player")) // ERROR cannot use
