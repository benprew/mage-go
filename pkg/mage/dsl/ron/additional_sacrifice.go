package ron

func compileAdditionalSacrifice(n Node, rules []Node) (*Node, *sacrificeValuePlan, error) {
	if err := arity(n, 1); err != nil {
		return nil, nil, err
	}
	cost := n.Args[0]
	if cost.Name != "SacrificeAPermanent" && cost.Name != "Exile" {
		return nil, nil, unsupported(cost, "additional casting cost")
	}
	if _, _, err := compileCosts(cost); err != nil {
		return nil, nil, err
	}
	for _, rule := range rules {
		if rule.Name != "SpellActions" {
			continue
		}
		if err := arity(rule, 1); err != nil {
			return nil, nil, err
		}
		if cost.Name == "Exile" {
			return &cost, nil, nil
		}
		paid, err := compileSacrificeValuePlan(cost, rule.Args[0])
		return &cost, paid, err
	}
	return nil, nil, problem(n, "additional casting cost requires SpellActions")
}
