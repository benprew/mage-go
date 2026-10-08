package ron

type opponentEntryChoicePlan struct{}

func validateOpponentEntryChoice(n Node) error {
	if err := arity(n, 2); err != nil {
		return err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "ThisPermanent" {
		return unsupported(n.Args[0], "opponent entry source")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return err
	}
	if len(operations) != 1 || operations[0].Name != "ChooseAPlayer" {
		return problem(n, "opponent entry requires one player choice")
	}
	operation := operations[0]
	return validateOpponentEntryOperation(operation)
}

func validateOpponentEntryOperation(operation Node) error {
	if err := arity(operation, 1); err != nil {
		return err
	}
	if operation.Args[0].Kind != Symbol || operation.Args[0].Name != "Opponent" {
		return unsupported(operation.Args[0], "entry player filter")
	}
	return nil
}

func compileOpponentEntryDeclaration(rules []Node, permanent bool) (*opponentEntryChoicePlan, error) {
	var plan *opponentEntryChoicePlan
	for _, rule := range rules {
		if rule.Name != "AsPermanentEnters" || rule.Kind != Tuple || len(rule.Args) != 2 || rule.Args[1].Kind != List {
			continue
		}
		for _, operation := range rule.Args[1].Args {
			if operation.Name != "ChooseAPlayer" {
				continue
			}
			if !permanent {
				return nil, unsupported(rule, "opponent entry choice on a nonpermanent")
			}
			if err := validateOpponentEntryChoice(rule); err != nil {
				return nil, err
			}
			if plan != nil {
				return nil, problem(rule, "duplicate opponent entry choice")
			}
			plan = &opponentEntryChoicePlan{}
		}
	}
	return plan, nil
}
