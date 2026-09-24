package fixture

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.DestroyEachWithoutRegeneration(dsl.This)

// ERROR does not implement mage.PermanentSet
