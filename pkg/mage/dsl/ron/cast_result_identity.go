package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type castResultPlan struct{ name string }

func hasCastResultRead(nodes []Node) bool {
	for _, n := range nodes {
		if n.Name == "ThePermanentSpellResolvedThisWay" || hasCastResultRead(n.Args) {
			return true
		}
	}
	return false
}

func faceDownCastProducer(n Node) bool {
	return n.Name == "CastCardInHandWithoutPayingAsAFacedownCreatureSpell" || n.Name == "MayAction" && len(n.Args) == 1 && n.Args[0].Name == "CastCardInHandWithoutPayingAsAFacedownCreatureSpell"
}

func compileCastResultProducer(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	plan := &castResultPlan{name: "cast result at " + n.Position.String()}
	optional := n.Name == "MayAction"
	action := n
	if optional {
		action = n.Args[0]
	}
	producer, err := compileFaceDownCastInto(action, ctx, optional, plan)
	if err != nil {
		return nil, err
	}
	ctx.castResult = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.castResults = maps.Clone(bindings.castResults)
		if bindings.castResults == nil {
			bindings.castResults = make(map[*castResultPlan]*mage.DamageSourceBinding)
		}
		bindings.castResults[plan] = mage.BindDamageSource(plan.name)
		effects := make([]mage.Effect, 1, 1+len(body))
		effects[0] = producer(bindings)
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}
