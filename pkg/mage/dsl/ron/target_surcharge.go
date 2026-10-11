package ron

func compileTargetSurcharge(n Node) (int, error) {
	if err := arity(n, 1); err != nil {
		return 0, err
	}
	payment := n.Args[0]
	if payment.Name != "PayMana" {
		return 0, unsupported(payment, "target surcharge payment")
	}
	if err := arity(payment, 1); err != nil {
		return 0, err
	}
	parts, err := list(payment.Args[0])
	if err != nil {
		return 0, err
	}
	if len(parts) != 1 {
		return 0, problem(payment.Args[0], "target surcharge requires one positive generic mana amount")
	}
	generic := parts[0]
	if generic.Name != "ManaCostGeneric" {
		return 0, unsupported(generic, "target surcharge mana")
	}
	if err := arity(generic, 1); err != nil {
		return 0, err
	}
	amount := generic.Args[0]
	if amount.Kind != IntegerNode || amount.Integer <= 0 {
		return 0, problem(amount, "target surcharge requires a positive integer literal")
	}
	return amount.Integer, nil
}
