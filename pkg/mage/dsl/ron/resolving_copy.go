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
	retainAbility := false
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
			switch operation.Name {
			case "KeepColor":
				if operation.Kind != Symbol || keepColors {
					return nil, problem(operation, "supply one KeepColor symbol")
				}
				keepColors = true
			case "AddAbilityVariable":
				if err := arity(operation, 1); err != nil {
					return nil, err
				}
				ref := operation.Args[0]
				if ref.Kind != Symbol || ref.Name != "ThisAbility" || ctx.copyAbility == nil || retainAbility {
					return nil, problem(operation, "supply one bound ThisAbility reference")
				}
				retainAbility = true
			default:
				return nil, unsupported(operation, "resolving copy modification")
			}
		}
		if len(operations) == 0 {
			return nil, problem(modifications, "supply resolving copy modifications")
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		var options []mage.ResolvingCopyOption
		if retainAbility {
			options = append(options, mage.RetainCopyAbility(bindings.copyAbilities[ctx.copyAbility]))
		}
		if optional {
			return mage.MayCopyPermanent(ctx.player.build(bindings), recipient.build(bindings), subject.build(bindings), keepColors, options...)
		}
		return mage.CopyPermanent(recipient.build(bindings), subject.build(bindings), keepColors, options...)
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
