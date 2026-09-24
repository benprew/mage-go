package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.WhenEvent(dsl.SourceHasCounters(dsl.Tide, 4), dsl.RemoveAllCounters(dsl.This, dsl.Tide))
// ERROR does not implement mage.EventPattern
