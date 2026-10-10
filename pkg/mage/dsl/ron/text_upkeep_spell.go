package ron

func upkeepSpellTextRules(rules []Node) bool {
	return len(rules) == 1 && upkeepObjectTextRules(rules)
}
