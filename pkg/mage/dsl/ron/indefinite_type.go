package ron

import (
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
	if len(parts) != 1 {
		return nil, problem(n.Args[1], "supply one indefinite type modification")
	}
	var modifications []func(instanceBindings) mage.ContinuousModification
	for _, part := range parts {
		switch part.Name {
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
