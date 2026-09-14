package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.WithCost(dsl.Sacrifice(dsl.This)) // ERROR Cost
