package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileDiscardReplacement(n Node, ctx scope) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if !ctx.permanent || ctx.sourceZone != nil {
		return nil, problem(n, "discard replacement requires a battlefield permanent")
	}
	event := n.Args[0]
	if event.Name != "AnEffectWouldCausePlayerToDiscardACard" {
		return nil, unsupported(event, "discard replacement event")
	}
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	if event.Args[0].Kind != Symbol || event.Args[0].Name != "You" {
		return nil, unsupported(event.Args[0], "discard replacement player")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Name != "MayAction" {
		return nil, problem(n.Args[1], "supply one optional discard destination")
	}
	optional := operations[0]
	if err := arity(optional, 1); err != nil {
		return nil, err
	}
	operation := optional.Args[0]
	if operation.Kind != Symbol || operation.Name != "DiscardItToTopOfLibraryInstead" {
		return nil, unsupported(operation, "discard replacement operation")
	}
	return func() mage.CardOption { return mage.WithStaticAbility(mage.DiscardToLibraryReplacement()) }, nil
}
