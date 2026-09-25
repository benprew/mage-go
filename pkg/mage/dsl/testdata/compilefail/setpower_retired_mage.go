package compilefail
import "github.com/benprew/mage-go/pkg/mage"
var _ = mage.SetPowerUntilEndOfTurn
// ERROR undefined: mage.SetPowerUntilEndOfTurn
