package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileTemporaryLayer(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	duration, err := compileTemporaryDuration(n.Args[2], ctx)
	if err != nil {
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
	var modifications []func(instanceBindings) mage.ContinuousModification
	for _, part := range parts {
		if part.Name == "AddAbilityVariable" {
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			if part.Args[0].Kind != Symbol || part.Args[0].Name != "TheChosenAbility" || ctx.chosenAbility == nil {
				return nil, problem(part.Args[0], "unbound or incompatible chosen ability")
			}
			ability := *ctx.chosenAbility
			modifications = append(modifications, func(bindings instanceBindings) mage.ContinuousModification {
				return ability.modification(subject.build(bindings))
			})
			continue
		}
		if part.Name == "SetPower" {
			build, err := compileTemporaryPower(part, subject)
			if err != nil {
				return nil, err
			}
			modifications = append(modifications, build)
			continue
		}
		if part.Name == "LosesAbility" {
			build, err := compileTemporaryKeywordRemoval(part, subject)
			if err != nil {
				return nil, err
			}
			modifications = append(modifications, build)
			continue
		}
		if part.Name == "SetCardtypes" {
			builds, err := compileTemporaryArtifactCreature(part, subject)
			if err != nil {
				return nil, err
			}
			modifications = append(modifications, builds...)
			continue
		}
		if part.Name == "SetPowerAndToughnessBoth" {
			build, err := compileTemporaryEqualPT(part, ctx, subject)
			if err != nil {
				return nil, err
			}
			modifications = append(modifications, build)
			continue
		}
		if part.Name == "SetPT" || part.Name == "AddCardtype" || part.Name == "AddCreatureType" || part.Name == "SetCreatureType" {
			var build func(instanceBindings) mage.ContinuousModification
			var err error
			if part.Name == "SetPT" && part.Kind == Tuple && len(part.Args) == 1 && part.Args[0].Name == "ManualPT" {
				build, err = compileTemporaryManualPT(part.Args[0], ctx, subject)
			} else {
				build, err = compileTemporaryType(part, subject)
			}
			if err != nil {
				return nil, err
			}
			modifications = append(modifications, build)
			continue
		}
		if part.Name == "AddAbility" {
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			abilities, err := list(part.Args[0])
			if err != nil {
				return nil, err
			}
			for _, node := range abilities {
				ability, err := compileTemporaryAbility(node)
				if err != nil {
					return nil, err
				}
				modifications = append(modifications, func(bindings instanceBindings) mage.ContinuousModification {
					return ability.modification(subject.build(bindings))
				})
			}
			continue
		}
		if part.Name == "AdjustPTX" {
			build, err := compileTemporaryPTX(part, ctx, subject)
			if err != nil {
				return nil, err
			}
			modifications = append(modifications, build)
			continue
		}
		if part.Name != "AdjustPT" {
			return nil, unsupported(part, "temporary layer action")
		}
		power, toughness, err := compilePT(part)
		if err != nil {
			return nil, err
		}
		modifications = append(modifications, func(bindings instanceBindings) mage.ContinuousModification {
			return mage.ModifyPT(subject.build(bindings), mage.Fixed(power), mage.Fixed(toughness))
		})
	}
	if len(modifications) == 0 {
		return nil, problem(n.Args[1], "supply at least one temporary modification")
	}
	return func(bindings instanceBindings) mage.Effect {
		values := make([]mage.ContinuousModification, len(modifications))
		for i, build := range modifications {
			values[i] = build(bindings)
		}
		return duration(bindings, values)
	}, nil
}
