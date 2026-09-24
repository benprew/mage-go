package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.ActivateModes(dsl.Costs(), dsl.WithTargets(dsl.TargetPlayer("player")))

// ERROR cannot use
