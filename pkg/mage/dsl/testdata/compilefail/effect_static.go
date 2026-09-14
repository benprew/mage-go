package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.WithStaticAbility(dsl.Sacrifice(dsl.This)) // ERROR ContinuousEffect
