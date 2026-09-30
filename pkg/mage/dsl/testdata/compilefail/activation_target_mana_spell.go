package fixture
import . "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = Spell(Draw(You, Fixed(1)), ActivationManaPerTarget(Red, 1))
// ERROR does not implement mage.SpellOption
