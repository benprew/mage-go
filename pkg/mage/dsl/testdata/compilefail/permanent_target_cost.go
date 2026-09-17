package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.TargetPermanent("creature", dsl.IsCreature, dsl.TapCost(dsl.This))
// ERROR does not implement mage.PermanentTargetOption
