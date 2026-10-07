package ron

import "github.com/benprew/mage-go/pkg/mage"

type abilityTargetPlan struct{ name string }

func compileAbilityTarget(n Node, ctx scope) (targetFactory, string, error) {
	filter := n.Args[0]
	if filter.Name != "And" {
		return nil, "", unsupported(filter, "activated ability target filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, "", err
	}
	parts, err := list(filter.Args[0])
	if err != nil {
		return nil, "", err
	}
	if len(parts) != 2 {
		return nil, "", problem(filter, "supply one activated ability filter and one source type")
	}
	activated := false
	var source Node
	for _, part := range parts {
		switch part.Name {
		case "ActivatedAbility":
			if _, err := symbol(part); err != nil {
				return nil, "", err
			}
			if activated {
				return nil, "", problem(part, "duplicate activated ability filter")
			}
			activated = true
		case "AbilityOfASource":
			if err := arity(part, 1); err != nil {
				return nil, "", err
			}
			if source.Name != "" {
				return nil, "", problem(part, "duplicate ability source filter")
			}
			source = part.Args[0]
		default:
			return nil, "", unsupported(part, "activated ability target filter")
		}
	}
	if !activated || source.Name != "IsCardtype" {
		return nil, "", problem(filter, "supply one activated ability filter and one source type")
	}
	if err := arity(source, 1); err != nil {
		return nil, "", err
	}
	kind, err := cardType(source.Args[0])
	if err != nil {
		return nil, "", err
	}
	return func(instanceBindings) mage.Target {
		return mage.DeclareActivatedAbilityTarget(ctx.targetNamespace+"Ref_TargetAbility", kind)
	}, "Ref_TargetAbility", nil
}

func compileAbilityRef(n Node, ctx scope) (*abilityTargetPlan, error) {
	if ctx.delayedStep {
		return nil, unsupported(n, "delayed ability reference")
	}
	if n.Kind != Symbol || ctx.abilityTarget == nil || n.Name != ctx.abilityTarget.name {
		return nil, problem(n, "unbound or incompatible ability reference")
	}
	return ctx.abilityTarget, nil
}
