package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileEntryHistory(n Node, ctx scope) (permanentSetFactory, error) {
	if !ctx.graveyardEnchant || !ctx.resolving || ctx.drawReplacementBody {
		return nil, problem(n, "entry history requires a resolving graveyard Aura collection")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "ThisPermanent" {
		return nil, unsupported(n.Args[0], "graveyard Aura entry source")
	}
	return func(instanceBindings) mage.PermanentSet {
		return mage.PermanentsEnteredWithSource(mage.This, mage.MatchingPermanents(mage.AnyPermanent))
	}, nil
}
