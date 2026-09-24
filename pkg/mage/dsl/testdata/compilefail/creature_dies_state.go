package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.WhenState(dsl.CreatureDies(), dsl.Draw(dsl.You, dsl.Fixed(1)))

// ERROR cannot use
