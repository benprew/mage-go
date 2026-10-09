package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCardNameCatalog(n Node) (*mage.CardNameCatalog, error) {
	if n.Kind != Symbol || n.Name != "AnyOracleCard" {
		return nil, unsupported(n, "card name catalog")
	}
	return mage.OracleCardNameCatalog(), nil
}

func compileCardNameChoice(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	catalog, err := compileCardNameCatalog(n.Args[0])
	if err != nil {
		return nil, err
	}
	player := ctx.player
	return func(bindings instanceBindings) mage.Effect {
		return mage.ChooseCardName(player.build(bindings), catalog, mage.BindCardName("chosen name "+n.Position.String()))
	}, nil
}
