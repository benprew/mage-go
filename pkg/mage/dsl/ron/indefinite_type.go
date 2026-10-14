package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileIndefiniteType(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	for _, part := range parts {
		if part.Name == "ReplaceColorWordVariableWithNewColorWordVariable" || part.Name == "ReplaceLandTypeVariableWithNewLandTypeVariable" {
			return compilePermanentObjectText(n, ctx, subject, parts)
		}
	}
	for _, part := range parts {
		if part.Name == "AddAbility" && len(part.Args) == 1 && part.Args[0].Kind == List {
			for _, ability := range part.Args[0].Args {
				if ability.Name == "If" || ability.Name == "TriggerA" {
					return compileLastingAbilityGrants(n, ctx, subject, parts)
				}
			}
		}
	}
	if len(parts) != 1 {
		return nil, problem(n.Args[1], "supply one indefinite type modification")
	}
	var modifications []func(instanceBindings) mage.ContinuousModification
	for _, part := range parts {
		switch part.Name {
		case "SetController":
			if ctx.enteringSource || ctx.drawReplacementBody {
				return nil, problem(part, "indefinite control requires an ordinary resolving body")
			}
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			controller, err := compilePlayerRef(part.Args[0], ctx)
			if err != nil {
				return nil, err
			}
			modifications = append(modifications, func(bindings instanceBindings) mage.ContinuousModification {
				return mage.ChangeController(subject.build(bindings), controller.build(bindings))
			})
		case "SetColor":
			colors, err := compileColorList(part)
			if err != nil {
				return nil, err
			}
			for i, color := range colors {
				if slices.Contains(colors[:i], color) {
					return nil, problem(part, "colors must be distinct")
				}
			}
			modifications = append(modifications, func(bindings instanceBindings) mage.ContinuousModification {
				return mage.SetColors(subject.build(bindings), colors...)
			})
		case "AddAbility":
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			abilities, err := list(part.Args[0])
			if err != nil {
				return nil, err
			}
			if len(abilities) == 0 {
				return nil, problem(part, "supply at least one ability")
			}
			if len(abilities) == 1 && (abilities[0].Name == "Activated" || abilities[0].Name == "ActivatedWithModifiers") {
				build, err := compileActivatedGrantModification(part, subject)
				if err != nil {
					return nil, err
				}
				modifications = append(modifications, build)
				continue
			}
			for _, ability := range abilities {
				if ability.Name == "PermanentLayerEffect" {
					return compileAcquiredStatic(n, ctx)
				}
			}
			for _, ability := range abilities {
				keyword, err := compileTemporaryKeyword(ability)
				if err != nil {
					return nil, err
				}
				modifications = append(modifications, func(bindings instanceBindings) mage.ContinuousModification {
					return mage.GrantKeywordModification(subject.build(bindings), keyword)
				})
			}
		case "SetToughness":
			build, err := compileIndefiniteToughness(part, ctx, subject)
			if err != nil {
				return nil, err
			}
			modifications = append(modifications, build)
		case "LosesAbility":
			build, err := compileTemporaryKeywordRemoval(part, subject)
			if err != nil {
				return nil, err
			}
			modifications = append(modifications, build)
		case "AddCardtype":
			build, err := compileTemporaryType(part, subject)
			if err != nil {
				return nil, err
			}
			modifications = append(modifications, build)
		case "SetLandType":
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			typeNode := part.Args[0]
			if typeNode.Kind != Symbol {
				return nil, unsupported(typeNode, "indefinite basic land type")
			}
			switch typeNode.Name {
			case "Plains", "Island", "Swamp", "Mountain", "Forest":
			default:
				return nil, unsupported(typeNode, "indefinite basic land type")
			}
			modifications = append(modifications, func(bindings instanceBindings) mage.ContinuousModification {
				return mage.SetSubtypeFamily(subject.build(bindings), mage.SubtypeLand, typeNode.Name)
			})
		default:
			return nil, unsupported(part, "indefinite type modification")
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		values := make([]mage.ContinuousModification, len(modifications))
		for i, build := range modifications {
			values[i] = build(bindings)
		}
		return mage.Until(core.Indefinite, values...)
	}, nil
}
