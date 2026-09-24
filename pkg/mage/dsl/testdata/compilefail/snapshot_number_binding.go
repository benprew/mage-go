package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.SnapshotNumber(dsl.Fixed(1), dsl.BindPermanent("permanent"))

// ERROR cannot use
