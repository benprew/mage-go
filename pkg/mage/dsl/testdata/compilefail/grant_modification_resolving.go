package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.Sequence(dsl.ActivatedAbilityModification(dsl.This, dsl.Costs(), dsl.Regenerate(dsl.This)))
// ERROR does not implement mage.ResolvingEffect
