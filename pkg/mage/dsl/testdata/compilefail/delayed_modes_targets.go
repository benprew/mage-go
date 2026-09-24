package fixture

import . "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = AtNextModes(EachEndStep, WithTargets(TargetPlayer("player")))

// ERROR does not implement mage.ModalDelayedStepOption
