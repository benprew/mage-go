package mage

// TargetOtherThanSavedPermanent excludes the exact visit in a saved binding.
// A new visit cannot inherit the exclusion.
func TargetOtherThanSavedPermanent(binding *PermanentBinding) PermanentTargetOption {
	return targetSavedExclusionOption{binding}
}

type targetSavedExclusionOption struct{ binding *PermanentBinding }

func (targetSavedExclusionOption) permanentTargetOption() {}
func (o targetSavedExclusionOption) applyPermanentTarget(t *PermanentTarget) {
	t.excludedBindings = append(t.excludedBindings, o.binding)
}
func (o targetSavedExclusionOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-saved-exclusion", children: []definitionEdge{{"reference", From(o.binding)}}}
}
