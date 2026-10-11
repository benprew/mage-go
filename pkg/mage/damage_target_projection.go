package mage

// DeclareOpponentChosenDamageTarget keeps target legality with the controller.
func DeclareOpponentChosenDamageTarget(name string) *DamageTargetBinding {
	return &DamageTargetBinding{&targetDeclaration{name: name, spec: TargetOpponentChoice(TargetDamageAnyTarget())}}
}

// PermanentFromDamageTarget keeps the target's saved permanent identity.
func PermanentFromDamageTarget(target *DamageTargetBinding) *PermanentTargetBinding {
	if target == nil {
		return nil
	}
	return &PermanentTargetBinding{target.targetDeclaration}
}
