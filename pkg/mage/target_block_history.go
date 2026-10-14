package mage

// TargetBlockedByThisTurn requires a block pair with an earlier singular target.
// The saved blocker visit can supply history after departure or phasing.
func TargetBlockedByThisTurn(reference *PermanentTargetBinding) PermanentTargetOption {
	return targetBlockHistoryDependency{reference}
}

type targetBlockHistoryDependency struct{ reference *PermanentTargetBinding }

func (targetBlockHistoryDependency) permanentTargetOption() {}
func (o targetBlockHistoryDependency) applyPermanentTarget(t *PermanentTarget) {
	t.blockHistoryDependencies = append(t.blockHistoryDependencies, o)
}
func (o targetBlockHistoryDependency) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-block-history-dependency", children: []definitionEdge{{"reference", o.reference}}}
}
func (o targetBlockHistoryDependency) matches(p *Permanent, ctx *EffectContext) bool {
	if o.reference == nil || o.reference.targetDeclaration == nil || p == nil || p.PhasedOut {
		return false
	}
	blocker, ok := o.reference.informationIdentity(ctx)
	return ok && ctx.Game.wasBlockedByThisTurn(blocker, p)
}
