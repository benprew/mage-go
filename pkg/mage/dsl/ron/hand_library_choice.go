package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileHandLibraryChoice(n Node, ctx scope) ([]effectFactory, error) {
	if n.Kind != Symbol {
		return nil, problem(n, "PutACardFromHandOnTopOfLibrary requires a symbol")
	}
	if !ctx.resolving || ctx.drawReplacementBody {
		return nil, problem(n, "hand-to-library choice requires an ordinary resolving body")
	}
	return []effectFactory{func(bindings instanceBindings) mage.Effect {
		return mage.PutChosenHandCardOnLibrary(ctx.player.build(bindings))
	}}, nil
}
