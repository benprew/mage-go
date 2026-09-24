package mage

// RegeneratesAttachment reports a direct regeneration instruction for the source's
// attachment. It follows ordered captures and snapshots without executing them.
// Unsupported instructions stop inspection. This query does not check costs,
// targets, attachment presence, or whether regeneration can prevent destruction.
func RegeneratesAttachment(ability ActivatedAbility) bool {
	attached := make(map[*PermanentBinding]bool)
	isAttached := func(subject PermanentRef) bool {
		switch ref := subject.(type) {
		case attachedPermanentRef:
			return true
		case savedPermanentRef:
			return attached[ref.binding]
		default:
			return false
		}
	}
	if action, ok := ability.(*ActionDefinition); ok {
		for _, capture := range action.dslCaptures {
			if capture, ok := capture.(activationPermanentCapture); ok {
				attached[capture.binding] = isAttached(capture.subject)
			}
		}
	}
	var inspect func(Effect) (found, supported bool)
	inspect = func(effect Effect) (bool, bool) {
		switch effect := effect.(type) {
		case *subjectAction:
			_, regenerates := effect.effect.(*regenerateTargetEffect)
			return regenerates && isAttached(effect.permanent), regenerates
		case *permanentSnapshot:
			attached[effect.binding] = isAttached(effect.subject)
			return false, true
		case *sequenceEffect:
			for _, child := range effect.effects {
				found, supported := inspect(child)
				if found || !supported {
					return found, supported
				}
			}
			return false, true
		default:
			return false, false
		}
	}
	for _, effect := range ability.Effects() {
		found, supported := inspect(effect)
		if found || !supported {
			return found
		}
	}
	return false
}
