package ron

func validateColorEntryOperation(n Node) error {
	if err := arity(n, 1); err != nil {
		return err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "AnyColor" {
		return unsupported(n.Args[0], "entry color filter")
	}
	return nil
}

func validateColorEntryDeclaration(rules []Node, permanent bool) error {
	entries, choices := 0, 0
	var choiceRule Node
	for _, rule := range rules {
		if rule.Name != "AsPermanentEnters" {
			continue
		}
		entries++
		if rule.Kind != Tuple || len(rule.Args) != 2 || rule.Args[1].Kind != List {
			continue
		}
		for _, operation := range rule.Args[1].Args {
			if operation.Name != "ChooseAColor" {
				continue
			}
			if !permanent {
				return unsupported(rule, "color entry choice on a nonpermanent")
			}
			if rule.Args[0].Kind != Symbol || rule.Args[0].Name != "ThisPermanent" || len(rule.Args[1].Args) != 1 {
				return problem(rule, "color entry requires one source choice")
			}
			if err := validateColorEntryOperation(operation); err != nil {
				return err
			}
			choices++
			choiceRule = rule
			if choices > 1 {
				return problem(rule, "duplicate color entry choice")
			}
		}
	}
	if choices != 0 && entries != 1 {
		return problem(choiceRule, "color entry choice requires one entry declaration")
	}
	return nil
}

func hasColorEntryChoice(rules []Node) bool {
	for _, rule := range rules {
		if rule.Name == "AsPermanentEnters" && len(rule.Args) == 2 && rule.Args[1].Kind == List {
			for _, operation := range rule.Args[1].Args {
				if operation.Name == "ChooseAColor" {
					return true
				}
			}
		}
	}
	return false
}
