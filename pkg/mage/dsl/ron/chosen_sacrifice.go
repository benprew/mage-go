package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileChosenSacrifice(n Node, ctx scope) (effectFactory, error) {
	return compileChosenSacrificeWithResult(n, ctx, nil)
}

func compileChosenSacrificeWithResult(n Node, ctx scope, result *sacrificeSubtypePlan) (effectFactory, error) {
	count := 1
	if n.Name == "SacrificeAPermanentOfAPlayersChoice" {
		count = 2
	}
	if err := arity(n, count); err != nil {
		return nil, err
	}
	subjects, err := compileContextualCollection(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	player := ctx.player
	chooser := player
	opponent := false
	if count == 2 {
		if n.Args[1].Kind == Symbol && n.Args[1].Name == "Opponent" {
			opponent = true
		} else {
			chooser, err = compilePlayerRef(n.Args[1], ctx)
			if err != nil {
				return nil, err
			}
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		var options []mage.SacrificeResultOption
		if result != nil && result.subtype != "" {
			options = append(options, mage.SacrificeSubtypeInto(result.subtype, sacrificeSubtypeBinding(bindings, result)))
		}
		sacrificing := player.build(bindings)
		if !opponent {
			return mage.SacrificeChosenBy(sacrificing, chooser.build(bindings), subjects(bindings), options...)
		}
		chosen := mage.BindPlayer("sacrifice chooser " + n.Position.String())
		return mage.Sequence(mage.ChoosePlayer(sacrificing, mage.OpponentsOf(sacrificing), chosen), mage.SacrificeChosenBy(sacrificing, mage.FromPlayer(chosen), subjects(bindings), options...))
	}, nil
}
