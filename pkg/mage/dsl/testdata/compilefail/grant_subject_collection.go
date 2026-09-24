package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.GrantActivatedAbility(dsl.MatchingPermanents(dsl.IsCreature), dsl.Costs(), dsl.Regenerate(dsl.This))
// ERROR does not implement mage.PermanentRef
