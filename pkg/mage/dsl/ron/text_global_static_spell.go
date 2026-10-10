package ron

func globalStaticSpellTextRules(rules []Node) bool {
	if len(rules) != 1 {
		return false
	}
	rule := rules[0]
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
