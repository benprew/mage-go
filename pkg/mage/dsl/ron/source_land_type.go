package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSourceLandType(n Node, ctx scope) (effectFactory, error) {
	if !ctx.permanent || !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "source-bound land type requires an ordinary resolving permanent body")
	}
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	duration := n.Args[2]
	if err := arity(duration, 1); err != nil {
		return nil, err
	}
	if duration.Args[0].Kind != Symbol || duration.Args[0].Name != "ThisPermanent" {
		return nil, unsupported(duration.Args[0], "land type duration source")
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) != 1 || parts[0].Name != "SetLandType" {
		return nil, problem(n.Args[1], "source-bound land type requires one SetLandType operation")
	}
	modification, err := compileBasicLandTypeModification(parts[0], subject)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.UntilSourceLeaves(mage.This, modification(bindings))
	}, nil
}
