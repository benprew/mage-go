package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.PreventNextDamage(dsl.MatchingPermanents(dsl.IsCreature), dsl.Fixed(2))
// ERROR does not implement mage.DamageRecipientRef
