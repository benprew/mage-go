package compilefail
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.CantBeBlocked(dsl.You)
// ERROR does not implement
