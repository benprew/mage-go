package compilefail

import "github.com/benprew/mage-go/pkg/mage/dsl"

var _ = dsl.ActivatedModesModification(dsl.This, dsl.Costs(), dsl.WithTargets(dsl.TargetPlayer("player")))

// ERROR ModalActivationOption
