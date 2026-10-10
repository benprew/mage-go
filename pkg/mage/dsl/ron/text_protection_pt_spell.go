package ron

func protectionPTTextRules(rules []Node) bool {
	if len(rules) != 2 {
		return false
	}
	for i, rule := range rules {
		if rule.Kind != Tuple || rule.Name != "Protection" || len(rule.Args) != 1 {
			continue
		}
		colors, err := protectionColors(rule.Args[0])
		if err != nil || len(colors) == 0 {
			return false
		}
		conditional := rules[1-i]
		if conditional.Kind != Tuple || conditional.Name != "If" || len(conditional.Args) != 2 {
			return false
		}
		condition := conditional.Args[0]
		if condition.Kind != Tuple || condition.Name != "APlayerPassesFilter" || len(condition.Args) != 2 {
			return false
		}
		selector := condition.Args[1]
		if selector.Kind != Tuple || selector.Name != "ControlsA" || len(selector.Args) != 1 || selector.Args[0].Name != "And" {
			return false
		}
		return compileOpponentPTRevision(conditional) != nil
	}
	return false
}
