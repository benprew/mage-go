package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileDeathReturnHand(n Node, ctx scope) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if !ctx.permanent || ctx.sourceZone != nil {
		return nil, problem(n, "death return requires a battlefield permanent")
	}
	event := n.Args[0]
	if err := deathReturnUnary(event, "APermanentWouldDie"); err != nil {
		return nil, err
	}
	collection := event.Args[0]
	if err := deathReturnUnary(collection, "SinglePermanent"); err != nil {
		return nil, err
	}
	if err := deathReturnSymbol(collection.Args[0], "ThisPermanent"); err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 2 {
		return nil, problem(n.Args[1], "supply a hand return and one next-turn hand restriction")
	}
	if err := deathReturnSymbol(operations[0], "PutItInOwnersHandInstead"); err != nil {
		return nil, err
	}
	playerEffect := operations[1]
	if playerEffect.Kind != Tuple || playerEffect.Name != "CreatePlayerEffectUntil" {
		return nil, unsupported(playerEffect, "death return continuation")
	}
	if err := arity(playerEffect, 3); err != nil {
		return nil, err
	}
	if err := deathReturnSymbol(playerEffect.Args[0], "OwnerOfTheCardReturnedToHandThisWay"); err != nil {
		return nil, err
	}
	rules, err := list(playerEffect.Args[1])
	if err != nil {
		return nil, err
	}
	if len(rules) != 2 {
		return nil, problem(playerEffect.Args[1], "supply one reveal and one play restriction")
	}
	seen := make(map[string]bool)
	for _, rule := range rules {
		if rule.Name != "PlaysWithCardInHandRevealed" && rule.Name != "CantPlayCardInHand" || seen[rule.Name] {
			return nil, unsupported(rule, "returned hand restriction")
		}
		if err := deathReturnUnary(rule, rule.Name); err != nil {
			return nil, err
		}
		if err := deathReturnSymbol(rule.Args[0], "TheCardReturnedToHandThisWay"); err != nil {
			return nil, err
		}
		seen[rule.Name] = true
	}
	duration := playerEffect.Args[2]
	if err := deathReturnUnary(duration, "UntilPlayersNextTurn"); err != nil {
		return nil, err
	}
	if err := deathReturnSymbol(duration.Args[0], "OwnerOfTheCardReturnedToHandThisWay"); err != nil {
		return nil, err
	}
	return func() mage.CardOption { return mage.WithAbility(mage.ReturnToHandInsteadOfDyingWithPlayLock()) }, nil
}

func deathReturnUnary(n Node, name string) error {
	if n.Kind != Tuple || n.Name != name {
		return unsupported(n, "death return declaration")
	}
	return arity(n, 1)
}

func deathReturnSymbol(n Node, name string) error {
	if n.Kind != Symbol || n.Name != name {
		return unsupported(n, "death return reference")
	}
	return nil
}
