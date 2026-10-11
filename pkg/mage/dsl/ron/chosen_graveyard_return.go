package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileChosenGraveyardReturn(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	filter := n.Args[0]
	if filter.Name != "IsCardtype" {
		return nil, unsupported(filter, "graveyard return choice filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	if filter.Args[0].Kind != Symbol || filter.Args[0].Name != "Creature" {
		return nil, unsupported(filter.Args[0], "graveyard return choice card type")
	}
	graveyard, err := compilePlayerRef(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	entries, err := list(n.Args[2])
	if err != nil {
		return nil, err
	}
	if len(entries) != 1 {
		return nil, problem(n.Args[2], "supply one graveyard return entry operation")
	}
	entry := entries[0]
	switch entry.Name {
	case "EntersUnderOwnersControl":
		if _, err := symbol(entry); err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.Effect {
			return mage.ReturnChosenGraveyardCardToOwner(graveyard.build(bindings), mage.IsCreatureCard)
		}, nil
	case "EntersUnderPlayersControl":
		if err := arity(entry, 1); err != nil {
			return nil, err
		}
		controller, err := compilePlayerRef(entry.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.Effect {
			return mage.ReturnChosenGraveyardCard(graveyard.build(bindings), controller.build(bindings), mage.IsCreatureCard)
		}, nil
	default:
		return nil, unsupported(entry, "graveyard return entry operation")
	}
}
