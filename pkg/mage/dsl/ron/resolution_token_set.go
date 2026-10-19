package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type resolutionTokensPlan struct{ name string }

func hasResolutionTokensRead(nodes []Node) bool {
	for _, n := range nodes {
		if n.Name == "TheTokensCreatedThisWay" || hasResolutionTokensRead(n.Args) {
			return true
		}
		for _, child := range n.Fields {
			if hasResolutionTokensRead([]Node{child}) {
				return true
			}
		}
	}
	return false
}

func compileResolutionTokens(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.drawReplacementBody || ctx.createdTokenLink != nil {
		return nil, problem(n, "saved token results require an ordinary resolving body without a persistent token link")
	}
	plan := &resolutionTokensPlan{name: "created tokens " + n.Position.String()}
	ctx.resolutionTokens = plan
	creation, err := compileTokenCreationWithResult(n, ctx, plan)
	if err != nil {
		return nil, err
	}
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.resolutionTokens = maps.Clone(bindings.resolutionTokens)
		if bindings.resolutionTokens == nil {
			bindings.resolutionTokens = make(map[*resolutionTokensPlan]*mage.CreatedTokensBinding)
		}
		bindings.resolutionTokens[plan] = mage.BindCreatedTokens(plan.name)
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, creation(bindings))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}

func compileResolutionTokensRef(n Node, ctx scope) (permanentSetFactory, error) {
	if n.Kind != Symbol || ctx.resolutionTokens == nil || !ctx.resolving || ctx.drawReplacementBody {
		return nil, problem(n, "created token result requires a preceding producer in this resolving scope")
	}
	return func(bindings instanceBindings) mage.PermanentSet {
		return mage.FromCreatedTokens(bindings.resolutionTokens[ctx.resolutionTokens])
	}, nil
}

func compileDelayedTokenSetLeave(n Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.drawReplacementBody || ctx.sourceZone != nil {
		return nil, problem(n, "delayed token departure requires an ordinary resolving body")
	}
	event := n.Args[0]
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	if event.Args[0].Name != "TheTokensCreatedThisWay" {
		return nil, unsupported(event.Args[0], "delayed token departure subject")
	}
	subjects, err := compileResolutionTokensRef(event.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	body := n.Args[1]
	if body.Name != "ActionList" {
		return nil, unsupported(body, "delayed token departure body")
	}
	if err := arity(body, 1); err != nil {
		return nil, err
	}
	parts, err := list(body.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(body, "delayed token departure requires at least one action")
	}
	ctx.delayedStep = true
	actions, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		effects := make([]mage.Effect, len(actions))
		for i, build := range actions {
			effects[i] = build(bindings)
		}
		return mage.WhenAnyLeaves(subjects(bindings), mage.Sequence(effects...))
	}, nil
}
