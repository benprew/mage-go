package ron

func globalGrantSpellTextRules(rules []Node) bool {
	if len(rules) != 1 {
		return false
	}
	rule := rules[0]
	if rule.Kind != Tuple || rule.Name != "EachPermanentLayerEffect" || len(rule.Args) != 2 || rule.Args[1].Kind != List || len(rule.Args[1].Args) == 0 {
		return false
	}
	hasGrant := false
	for _, part := range rule.Args[1].Args {
		switch part.Name {
		case "AdjustPT":
			if _, _, err := compilePT(part); err != nil {
				return false
			}
		case "AddAbility":
			if part.Kind != Tuple || len(part.Args) != 1 || part.Args[0].Kind != List || len(part.Args[0].Args) == 0 {
				return false
			}
			for _, ability := range part.Args[0].Args {
				if ability.Name == "Landwalk" {
					if _, err := compileLandwalkKeyword(ability); err != nil {
						return false
					}
					continue
				}
				if _, ok := keywords[ability.Name]; !ok || ability.Kind != Symbol {
					return false
				}
			}
			hasGrant = true
		default:
			return false
		}
	}
	return hasGrant
}
