package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.ChooseRandomPermanent(dsl.This, dsl.BindPermanent("chosen"))
// ERROR does not implement mage.PermanentSet
