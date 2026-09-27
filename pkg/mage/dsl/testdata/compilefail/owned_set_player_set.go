package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.PermanentsOwnedBy(dsl.EachPlayer, dsl.IsArtifact)

// ERROR does not implement mage.PlayerRef
