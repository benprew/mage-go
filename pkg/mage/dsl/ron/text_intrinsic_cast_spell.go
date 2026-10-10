package ron

func intrinsicCastSpellTextRules(rules []Node) bool {
	intrinsic := make([]Node, 0, len(rules))
	events := 0
	for _, rule := range rules {
		if spellCastSpellTextRules([]Node{rule}) {
			events++
		} else {
			intrinsic = append(intrinsic, rule)
		}
	}
	return events == 1 && intrinsicCreatureSpellTextRules(intrinsic)
}
