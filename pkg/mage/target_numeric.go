package mage

// TargetToughnessLessThanSourcePower checks current target toughness against
// source power. An absent source uses its retained power at resolution.
func TargetToughnessLessThanSourcePower() PermanentTargetOption {
	return targetToughnessLessThanSourcePowerOption{}
}

type targetToughnessLessThanSourcePowerOption struct{}

func (targetToughnessLessThanSourcePowerOption) permanentTargetOption() {}
func (targetToughnessLessThanSourcePowerOption) applyPermanentTarget(t *PermanentTarget) {
	t.toughnessLessThanSourcePower = true
}
func (targetToughnessLessThanSourcePowerOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-source-power", Text: "toughness below source power", battlefieldSource: true}
}

func targetSourcePower(card Card, g *Game) (int, bool) {
	if card == nil {
		return 0, false
	}
	if retained, ok := card.(targetSourceLKI); ok && retained.powerKnown {
		return retained.power, true
	}
	if p := g.FindPermanent(card.ID()); p != nil && !p.PhasedOut {
		return p.CurrentPower(g), true
	}
	return 0, false
}
