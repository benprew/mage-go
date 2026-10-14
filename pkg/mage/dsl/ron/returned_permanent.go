package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type returnedPermanentPlan struct{ label string }

func hasReturnedPermanentRead(nodes []Node) bool {
	for _, n := range nodes {
		if n.Name == "ThePermanentPutOnTheBattlefieldThisWay" || hasReturnedPermanentRead(n.Args) {
			return true
		}
	}
	return false
}

func compileReturnedPermanentProducer(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "returned permanent results require an ordinary resolving body")
	}
	plan := &returnedPermanentPlan{label: "returned permanent at " + n.Position.String()}
	producer, err := compileGraveyardReturnInto(n, ctx, plan)
	if err != nil {
		return nil, err
	}
	ctx.returnedPermanent = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.returnedPermanents = maps.Clone(bindings.returnedPermanents)
		if bindings.returnedPermanents == nil {
			bindings.returnedPermanents = make(map[*returnedPermanentPlan]*mage.PermanentBinding)
		}
		bindings.returnedPermanents[plan] = mage.BindPermanent(plan.label)
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, producer(bindings))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}
