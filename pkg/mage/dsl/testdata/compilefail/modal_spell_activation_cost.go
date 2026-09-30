package fixture
import . "github.com/benprew/mage-go/pkg/mage/dsl"
import "github.com/benprew/mage-go/pkg/mage/core"
var _ = SpellModesWithOptions([]ModalSpellOption{ActivationManaPerTarget(core.Red, 1)}, ModeOf("A", Draw(You, Fixed(1))), ModeOf("B", Draw(You, Fixed(2))))
// ERROR does not implement mage.ModalSpellOption
