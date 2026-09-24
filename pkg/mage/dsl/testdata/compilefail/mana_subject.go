package fixture
import (
 "github.com/benprew/mage-go/pkg/mage/dsl"
 "github.com/benprew/mage-go/pkg/mage/core"
)
var _ = dsl.AddManaAmount(core.Black, dsl.You)
// ERROR does not implement mage.ResolutionValue
