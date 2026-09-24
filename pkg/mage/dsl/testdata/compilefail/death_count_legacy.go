package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ dsl.ValueSource = dsl.CreaturesDiedThisTurn()

// ERROR does not implement mage.ValueSource
