package fixture
import . "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = SpellModesWithOptions([]ModalSpellOption{Draw(You, Fixed(1))}, ModeOf("A", Draw(You, Fixed(1))), ModeOf("B", Draw(You, Fixed(2))))
// ERROR does not implement mage.ModalSpellOption
