package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.SnapshotNumber(dsl.This, dsl.BindNumber("number"))

// ERROR does not implement mage.ResolutionValue
