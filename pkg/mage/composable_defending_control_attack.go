package mage

type defendingPlayerControlsAttackRule struct {
	filter PermanentFilter
}

// CanAttackOnlyIfDefendingPlayerControls checks the defender at attacker declaration.
func CanAttackOnlyIfDefendingPlayerControls(filter PermanentFilter) StaticRule {
	return defendingPlayerControlsAttackRule{filter: filter}
}

func (defendingPlayerControlsAttackRule) staticRule() {}

func (r defendingPlayerControlsAttackRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{PreventFromAttackingIfDefendingPlayerControls(r.filter)}
}

func (r defendingPlayerControlsAttackRule) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-attack-restriction", Text: "can attack only if defending player controls matching permanent"}
}
