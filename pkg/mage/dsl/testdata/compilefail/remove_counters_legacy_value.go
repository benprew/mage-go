package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var amount dsl.ValueSource = dsl.Fixed(1)
var _ = dsl.RemoveCountersFrom(dsl.This, dsl.Charge, amount)
// ERROR does not implement mage.ResolutionValue
