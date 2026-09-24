package fixture
import "github.com/benprew/mage-go/pkg/mage/dsl"
var _ = dsl.TurnStepBegins(dsl.Enters(dsl.This), dsl.BindPlayer("player")) // ERROR TurnStepPattern
