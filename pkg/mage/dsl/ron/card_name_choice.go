package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCardNameCatalog(n Node) (*mage.CardNameCatalog, error) {
	if n.Kind != Symbol || n.Name != "AnyOracleCard" {
		return nil, unsupported(n, "card name catalog")
	}
	return mage.OracleCardNameCatalog(), nil
}

type cardNameChoicePlan struct{ name string }
type cardNameRefFactory func(instanceBindings) mage.CardNameRef

func (p *cardNameChoicePlan) binding(bindings instanceBindings) *mage.CardNameBinding {
	if binding := bindings.chosenNames[p]; binding != nil {
		return binding
	}
	binding := mage.BindCardName(p.name)
	if bindings.chosenNames != nil {
		bindings.chosenNames[p] = binding
	}
	return binding
}

func compileCardNameRef(n Node, ctx scope) (cardNameRefFactory, error) {
	if n.Kind != Symbol || n.Name != "TheChosenName" || ctx.chosenName == nil {
		return nil, problem(n, "unbound or incompatible card name reference")
	}
	plan := ctx.chosenName
	return func(bindings instanceBindings) mage.CardNameRef { return mage.FromCardName(plan.binding(bindings)) }, nil
}

func compileCardNameChoiceWithResult(n Node, ctx scope) (effectFactory, *cardNameChoicePlan, error) {
	if err := arity(n, 1); err != nil {
		return nil, nil, err
	}
	catalog, err := compileCardNameCatalog(n.Args[0])
	if err != nil {
		return nil, nil, err
	}
	plan := &cardNameChoicePlan{name: "chosen name " + n.Position.String()}
	player := ctx.player
	return func(bindings instanceBindings) mage.Effect {
		return mage.ChooseCardName(player.build(bindings), catalog, plan.binding(bindings))
	}, plan, nil
}

func compileCardNameChoice(n Node, ctx scope) (effectFactory, error) {
	build, _, err := compileCardNameChoiceWithResult(n, ctx)
	return build, err
}
