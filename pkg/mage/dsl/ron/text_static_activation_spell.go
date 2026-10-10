package ron

func staticActivationSpellTextRules(rules []Node) bool {
	if len(rules) != 2 {
		return false
	}
	for i, rule := range rules {
		if !globalStaticSpellTextRules([]Node{rule}) {
			continue
		}
		if rule.Name == "EachPermanentLayerEffect" && rule.Args[1].Args[0].Name != "AdjustPT" {
			return false
		}
		action := rules[1-i]
		if action.Kind != Tuple || action.Name != "Activated" || len(action.Args) != 2 {
			return false
		}
		body := action.Args[1]
		return body.Name != "Modal_ChooseOne" && body.Name != "Targeted_Modal" && !conditionalManaBody(body) && !containsDeclarationName(body, "TimesThisAbilityHasBeenActivatedThisTurnIs")
	}
	return false
}
