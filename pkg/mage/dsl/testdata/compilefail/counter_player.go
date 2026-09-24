package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var amount dsl.ValueSource = dsl.Fixed(2)
var _ = dsl.AddCounters(dsl.You, dsl.P1P1, dsl.Fixed(1))
// ERROR does not implement mage.PermanentRef
