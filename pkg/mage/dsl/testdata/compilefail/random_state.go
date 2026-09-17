package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.WhenState(dsl.FlipCoin(dsl.You, dsl.OnWin(dsl.Sacrifice(dsl.This))), dsl.Sacrifice(dsl.This)) // ERROR StateCondition
