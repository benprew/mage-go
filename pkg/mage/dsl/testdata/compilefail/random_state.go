package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.WhenState(dsl.FlipCoinCond{}, dsl.Sacrifice(dsl.This)) // ERROR StateCondition
