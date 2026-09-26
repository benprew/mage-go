package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.ArtifactBecomesTapped().WithController(dsl.BindPermanent("artifact")) // ERROR *mage.PlayerBinding
