package ron

func upkeepSpellTextRules(rules []Node) bool {
	if !upkeepObjectTextRules(rules) {
		return false
	}
	for _, rule := range rules {
		if rule.Name == "AsPermanentEnters" {
			return false
		}
	}
	return true
}
