package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.ArtifactBecomesTapped().WithArtifact(dsl.BindPlayer("player")) // ERROR *mage.PermanentBinding
