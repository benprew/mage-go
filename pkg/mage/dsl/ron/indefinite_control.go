package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileIndefiniteControl(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "indefinite control requires an ordinary resolving body")
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	controller := ctx.player
	return func(bindings instanceBindings) mage.Effect {
		return mage.Until(core.Indefinite, mage.ChangeController(subject.build(bindings), controller.build(bindings)))
	}, nil
}

func compileDeferredMana(n Node, ctx scope) (optionFactory, bool, error) {
	if n.Name != "Activated" || len(n.Args) != 2 {
		return nil, false, nil
	}
	cost, body := n.Args[0], n.Args[1]
	if cost.Name != "TapPermanent" || len(cost.Args) != 1 || cost.Args[0].Kind != Symbol || cost.Args[0].Name != "ThisPermanent" || body.Name != "ActionList" || len(body.Args) != 1 || body.Args[0].Kind != List {
		return nil, false, nil
	}
	actions := body.Args[0].Args
	if len(actions) != 2 || actions[0].Name != "AddMana" || len(actions[0].Args) != 1 || actions[1].Name != "CreateFutureTrigger" {
		return nil, false, nil
	}
	productions, err := compileMana(actions[0].Args[0])
	if err != nil {
		return nil, true, err
	}
	if len(productions) != 1 || productions[0].Amount != 1 {
		return nil, false, nil
	}
	delayedBody := body
	delayedBody.Args = append([]Node(nil), body.Args...)
	delayedBody.Args[0].Args = append([]Node(nil), actions[1:]...)
	plan, err := compileActions(delayedBody, ctx)
	if err != nil {
		return nil, true, err
	}
	return func() mage.CardOption {
		built := plan.build()
		return mage.WithAbility(mage.NewManaAbility(productions[0].Color, built.effects...))
	}, true, nil
}
