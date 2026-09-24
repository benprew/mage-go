package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.GrantEachActivatedAbility(dsl.MatchingPermanents(dsl.IsCreature), dsl.Costs(), dsl.GrantEachKeyword(dsl.MatchingPermanents(dsl.IsCreature),dsl.Flying))
// ERROR does not implement mage.ResolvingEffect
