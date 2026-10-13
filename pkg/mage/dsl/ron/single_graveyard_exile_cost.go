package ron

func compileSingleGraveyardExileCost(n Node) (*exilePaymentPlan, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	count := n.Args[0]
	if count.Name != "Integer" {
		return nil, unsupported(count, "fixed single graveyard exile count")
	}
	if err := arity(count, 1); err != nil {
		return nil, err
	}
	if count.Args[0].Kind != IntegerNode || count.Args[0].Integer < 0 {
		return nil, problem(count, "single graveyard exile count requires a nonnegative integer")
	}
	filter, err := compileHandCardFilter(n.Args[1])
	if err != nil {
		return nil, err
	}
	players := n.Args[2]
	if players.Kind != Symbol || players.Name != "AnyPlayer" {
		return nil, unsupported(players, "single graveyard exile players")
	}
	return &exilePaymentPlan{filter: filter, singleGraveyard: true, amount: count.Args[0].Integer}, nil
}
