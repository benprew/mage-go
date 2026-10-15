package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileResolvingCopy(n Node, ctx scope, recipient permanentReferencePlan, parts []Node, optional bool) (effectFactory, error) {
	if ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "copy replacement requires an ordinary resolving body")
	}
	if len(parts) != 1 {
		return nil, problem(n, "supply one resolving copy modification")
	}
	part := parts[0]
	if err := arity(part, 2); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(part.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	keepColors := false
	modifications := part.Args[1]
	if modifications.Kind != Symbol || modifications.Name != "NoCopyEffects" {
		if modifications.Name != "CopyEffects" {
			return nil, unsupported(modifications, "resolving copy modifications")
		}
		if err := arity(modifications, 1); err != nil {
			return nil, err
		}
		operations, err := list(modifications.Args[0])
		if err != nil {
			return nil, err
		}
		for _, operation := range operations {
			if operation.Name != "KeepColor" {
				return nil, unsupported(operation, "resolving copy modification")
			}
		}
		if len(operations) != 1 || operations[0].Kind != Symbol || operations[0].Name != "KeepColor" {
			return nil, problem(modifications, "resolving copy supports one KeepColor symbol")
		}
		keepColors = true
	}
	return func(bindings instanceBindings) mage.Effect {
		if optional {
			return mage.MayCopyPermanent(ctx.player.build(bindings), recipient.build(bindings), subject.build(bindings), keepColors)
		}
		return mage.CopyPermanent(recipient.build(bindings), subject.build(bindings), keepColors)
	}, nil
}

func compileOptionalResolvingCopy(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	recipient, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) != 1 || parts[0].Name != "IsACopyOfPermanent" {
		return nil, problem(n, "optional layer action requires one resolving copy modification")
	}
	return compileResolvingCopy(n, ctx, recipient, parts, true)
}
