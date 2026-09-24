package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.Sequence(dsl.CreaturesDiedThisTurn())

// ERROR does not implement mage.ResolvingEffect
