package fixture
import (
 "github.com/benprew/mage-go/pkg/mage/dsl"
 "github.com/benprew/mage-go/pkg/mage/core"
)
var amount dsl.ValueSource = dsl.Fixed(2)
var _ = dsl.AddManaAmount(core.Black, amount)
// ERROR does not implement mage.ResolutionValue
