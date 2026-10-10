package ron

func copyEntrySpellTextRules(rules []Node) bool {
	if len(rules) != 1 {
		return false
	}
	rule := rules[0]
	if rule.Name != "AsPermanentEnters" || rule.Kind != Tuple || len(rule.Args) != 2 ||
		rule.Args[0].Kind != Symbol || rule.Args[0].Name != "ThisPermanent" || rule.Args[1].Kind != List || len(rule.Args[1].Args) != 1 {
		return false
	}
	optional := rule.Args[1].Args[0]
	if optional.Name != "MayActions" || optional.Kind != Tuple || len(optional.Args) != 1 || optional.Args[0].Kind != List || len(optional.Args[0].Args) != 1 {
		return false
	}
	operation := optional.Args[0].Args[0]
	if operation.Kind != Tuple || operation.Name != "EnterAsACopyOfAPermanent" || len(operation.Args) != 2 {
		return false
	}
	_, err := compileEntryCopy(operation)
	return err == nil
}
