package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileGraveyardBatchEntry(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	if !ctx.resolving || ctx.drawReplacementBody || ctx.enteringSource {
		return nil, problem(n, "mass graveyard entry requires a resolving action body")
	}
	filter := n.Args[0]
	if filter.Name != "IsCardtype" || filter.Kind != Tuple || len(filter.Args) != 1 || filter.Args[0].Kind != Symbol || filter.Args[0].Name != "Creature" {
		return nil, unsupported(filter, "mass graveyard entry filter")
	}
	if n.Args[1].Kind != Symbol || n.Args[1].Name != "AnyPlayer" {
		return nil, unsupported(n.Args[1], "mass graveyard entry players")
	}
	entries, err := list(n.Args[2])
	if err != nil {
		return nil, err
	}
	if len(entries) != 1 || entries[0].Kind != Symbol || entries[0].Name != "EntersNormally" {
		return nil, unsupported(n.Args[2], "mass graveyard entry operations")
	}
	return func(instanceBindings) mage.Effect {
		return mage.PutGraveyardCreaturesOntoBattlefieldUnderOwnersControl()
	}, nil
}
