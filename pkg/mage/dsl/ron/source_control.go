package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSourceControl(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if !ctx.permanent {
		return nil, problem(n, "source control duration requires a permanent source")
	}
	duration := n.Args[1]
	if duration.Name == "UntilPermanentLeavesBattlefield" {
		if err := arity(duration, 1); err != nil {
			return nil, err
		}
		if ctx.enteringSource || ctx.drawReplacementBody {
			return nil, problem(n, "source presence control requires an ordinary resolving body")
		}
		if duration.Args[0].Kind != Symbol || duration.Args[0].Name != "ThisPermanent" {
			return nil, unsupported(duration.Args[0], "source presence duration reference")
		}
		return func(bindings instanceBindings) mage.Effect {
			return mage.UntilSourcePresent(mage.This, mage.ChangeController(subject.build(bindings), mage.You))
		}, nil
	}
	tapped := false
	if duration.Name == "Or" {
		if err := arity(duration, 1); err != nil {
			return nil, err
		}
		parts, listErr := list(duration.Args[0])
		if listErr != nil {
			return nil, listErr
		}
		if len(parts) != 2 {
			return nil, problem(duration, "supply control and tapped duration endpoints")
		}
		duration = parts[0]
		endpoint := parts[1]
		if endpoint.Name != "ForAsLongAsPermanentRemainsTapped" {
			return nil, unsupported(endpoint, "source control tapped endpoint")
		}
		if err := arity(endpoint, 1); err != nil {
			return nil, err
		}
		if endpoint.Args[0].Kind != Symbol || endpoint.Args[0].Name != "ThisPermanent" {
			return nil, unsupported(endpoint.Args[0], "source control tapped reference")
		}
		tapped = true
	}
	if duration.Name != "UntilPermanentChangesControl" {
		return nil, unsupported(duration, "source control duration")
	}
	if err := arity(duration, 2); err != nil {
		return nil, err
	}
	if duration.Args[0].Kind != Symbol || duration.Args[0].Name != "You" {
		return nil, unsupported(duration.Args[0], "source control player")
	}
	if duration.Args[1].Kind != Symbol || duration.Args[1].Name != "ThisPermanent" {
		return nil, unsupported(duration.Args[1], "source control reference")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.GainControlWhileSourceControlled(subject.build(bindings), tapped)
	}, nil
}
