package ron

func spellCastSpellTextRules(rules []Node) bool {
	if len(rules) != 1 {
		return false
	}
	rule := rules[0]
	if rule.Kind != Tuple || rule.Name != "TriggerA" || len(rule.Args) != 2 {
		return false
	}
	event := rule.Args[0]
	return event.Kind == Tuple && event.Name == "WhenAPlayerCastsASpell" && len(event.Args) == 2 &&
		event.Args[0].Kind == Symbol && (event.Args[0].Name == "AnyPlayer" || event.Args[0].Name == "Opponent")
}
