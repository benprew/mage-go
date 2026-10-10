package ron

func globalStaticSpellTextRules(rules []Node) bool {
	if len(rules) != 1 {
		return false
	}
	rule := rules[0]
	if conditionalLandSourceSpellTextRule(rule) {
		return true
	}
	if rule.Kind != Tuple || rule.Name != "EachPermanentLayerEffect" || len(rule.Args) != 2 || rule.Args[1].Kind != List || len(rule.Args[1].Args) != 1 {
		return false
	}
	operation := rule.Args[1].Args[0]
	switch operation.Name {
	case "SetLandType":
		return operation.Kind == Tuple && len(operation.Args) == 1
	case "AdjustPT":
		return operation.Kind == Tuple && len(operation.Args) == 2 && requiresCreature(rule.Args[0])
	}
	return false
}

func conditionalLandSourceSpellTextRule(rule Node) bool {
	if rule.Kind != Tuple || rule.Name != "If" || len(rule.Args) != 2 {
		return false
	}
	condition := rule.Args[0]
	if len(condition.Args) != 2 {
		return false
	}
	selector := condition.Args[1]
	if selector.Name != "ControlsA" || len(selector.Args) != 1 || selector.Args[0].Name != "IsLandType" {
		return false
	}
	return compileConditionalSourcePTRevision(rule) != nil || compileOpponentPTRevision(rule) != nil
}
