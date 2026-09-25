package compilefail
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.SetBasePower(dsl.MatchingPermanents(dsl.IsCreature), dsl.Fixed(0))
// ERROR does not implement
