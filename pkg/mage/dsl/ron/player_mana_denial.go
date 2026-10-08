package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileLoseUnspentMana(n Node, ctx scope) ([]effectFactory, error) {
	if n.Kind != Symbol {
		return nil, problem(n, "LoseUnspentMana requires a symbol")
	}
	return []effectFactory{func(bindings instanceBindings) mage.Effect {
		return mage.LoseUnspentManaFor(ctx.player.build(bindings))
	}}, nil
}
