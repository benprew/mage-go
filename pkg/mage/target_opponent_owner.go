package mage

// TargetOpponentOwns restricts targets to permanents an opponent owns.
func TargetOpponentOwns() PermanentTargetOption { return targetOpponentOwnsOption{} }

type targetOpponentOwnsOption struct{}

func (targetOpponentOwnsOption) permanentTargetOption() {}
func (targetOpponentOwnsOption) applyPermanentTarget(t *PermanentTarget) {
	t.opponentOwned = true
}
func (targetOpponentOwnsOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-owner", Text: "an opponent owns"}
}
