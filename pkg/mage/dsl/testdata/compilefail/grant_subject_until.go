package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
import "github.com/benprew/mage-go/pkg/mage/core"
var _ = dsl.Until(core.EndOfTurn, dsl.GrantActivatedAbility(dsl.This, dsl.Costs(), dsl.Regenerate(dsl.This)))
// ERROR does not implement mage.ContinuousModification
