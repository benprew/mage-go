package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.Spell(dsl.Draw(dsl.You, dsl.Fixed(1)), dsl.OnlyOnYourTurn())
// ERROR does not implement mage.SpellOption
