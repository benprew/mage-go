package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.WhenState(dsl.EventSourceIsSelf{}, dsl.Sacrifice(dsl.This)) // ERROR StateCondition
