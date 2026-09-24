package fixture

import (
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

var _ = dsl.Sequence(dsl.SourceCounters(core.Charge))

// ERROR does not implement mage.ResolvingEffect
