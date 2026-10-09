package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileLibraryTopMove(n Node, ctx scope) ([]effectFactory, error) {
	if n.Kind != Symbol {
		return nil, problem(n, n.Name+" requires a symbol")
	}
	return []effectFactory{func(bindings instanceBindings) mage.Effect {
		player := ctx.player.build(bindings)
		if n.Name == "PutTopOfLibraryInHand" {
			return mage.PutLibraryTopInHand(player)
		}
		return mage.PutLibraryTopInGraveyard(player)
	}}, nil
}
