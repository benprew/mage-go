package fixture
import . "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = Static(Until(EndOfTurn,ModifyPT(This,Fixed(1),Fixed(1))))
// ERROR does not implement mage.StaticRule
