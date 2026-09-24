package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.ManaValueOf(dsl.MatchingPermanents(dsl.IsArtifact))
// ERROR does not implement mage.PermanentRef
