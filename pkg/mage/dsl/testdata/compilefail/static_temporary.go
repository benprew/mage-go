package fixture
import . "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = Until(EndOfTurn,While(Controls(IsCreature),ModifyPT(This,Fixed(1),Fixed(1))))
// ERROR does not implement mage.ContinuousModification
