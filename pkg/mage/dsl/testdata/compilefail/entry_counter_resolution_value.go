package fixture

import (
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

var _ = dsl.EntersWithCounters(core.P1P1, dsl.Fixed(1))

// ERROR does not implement mage.EntryCounterValue
