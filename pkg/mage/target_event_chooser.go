package mage

// TargetChosenBy uses a saved event player for target choice at trigger placement.
// Copies use their controller for new target choices.
func TargetChosenBy(player PlayerRef) PermanentTargetOption {
	return targetChosenByOption{player}
}

type targetChosenByOption struct{ player PlayerRef }

func (targetChosenByOption) permanentTargetOption() {}
func (o targetChosenByOption) applyPermanentTarget(t *PermanentTarget) {
	t.chooserRef = o.player
	t.chooserCount++
}
func (o targetChosenByOption) inspectDefinition() definitionNode {
	n := definitionNode{Role: "target-chooser-reference", children: []definitionEdge{{"player", o.player}}}
	if _, ok := o.player.(savedPlayerRef); !ok {
		n.problem = "target chooser: supply a saved event player"
	}
	return n
}

func targetChooserReference(target Target) PlayerRef {
	if declaration, ok := target.(*targetDeclaration); ok {
		target = declaration.spec
	}
	if permanent, ok := target.(*PermanentTarget); ok {
		return permanent.chooserRef
	}
	return nil
}
