package compilefail
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.SetBasePowerToughness(dsl.MatchingPermanents(dsl.IsCreature), dsl.Fixed(0), dsl.Fixed(2))
// ERROR does not implement
