package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.ActivatedAbilityModification(dsl.You, dsl.Costs(), dsl.Regenerate(dsl.This))
// ERROR does not implement mage.PermanentRef
