package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.GrantEachActivatedAbility(dsl.MatchingPermanents(dsl.IsCreature), dsl.Draw(dsl.You, dsl.Fixed(1)), dsl.Regenerate(dsl.This))
// ERROR cannot use
