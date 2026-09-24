package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var amount dsl.ValueSource = dsl.Fixed(2)
var _ = dsl.AddCounters(dsl.This, dsl.P1P1, dsl.Draw(dsl.You, dsl.Fixed(1)))
// ERROR does not implement mage.ResolutionValue
