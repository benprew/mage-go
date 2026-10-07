package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCounterChoice(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	options, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(options) < 2 {
		return nil, problem(n, "action choice requires at least two alternatives")
	}
	plans := make([]effectFactory, len(options))
	for i, option := range options {
		parts, err := list(option)
		if err != nil {
			return nil, err
		}
		if len(parts) != 1 || parts[0].Name != "PutCounters" {
			return nil, unsupported(option, "counter choice body")
		}
		action := parts[0]
		if err := arity(action, 1); err != nil {
			return nil, err
		}
		instructions, err := list(action.Args[0])
		if err != nil {
			return nil, err
		}
		if len(instructions) != 1 || instructions[0].Name != "ACounterOfTypeOnPermanent" {
			return nil, unsupported(action, "counter choice instruction")
		}
		instruction := instructions[0]
		if err := arity(instruction, 2); err != nil {
			return nil, err
		}
		if instruction.Args[0].Name != "PTCounter" || instruction.Args[1].Kind != Symbol || instruction.Args[1].Name != "Ref_TargetPermanent" {
			return nil, unsupported(instruction, "target P/T counter choice")
		}
		plans[i], err = compileCounterPlacement(action, ctx)
		if err != nil {
			return nil, err
		}
	}
	return buildResolutionChoice(plans, nil), nil
}

func compileAbilityChoice(n, continuation Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	options, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(options) < 2 {
		return nil, problem(n, "ability choice requires at least two alternatives")
	}
	if continuation.Name != "CreatePermanentLayerEffectUntil" {
		return nil, unsupported(continuation, "ability choice continuation")
	}
	if err := arity(continuation, 3); err != nil {
		return nil, err
	}
	modifications, err := list(continuation.Args[1])
	if err != nil {
		return nil, err
	}
	found := false
	for _, part := range modifications {
		if part.Name == "AddAbilityVariable" {
			found = true
		}
	}
	if !found {
		return nil, problem(continuation, "ability choice continuation must use TheChosenAbility")
	}
	plans := make([]effectFactory, len(options))
	labels := make([]string, len(options))
	seen := map[temporaryAbilityPlan]bool{}
	for i, option := range options {
		ability, err := compileTemporaryAbility(option)
		if err != nil {
			return nil, err
		}
		if seen[ability] {
			return nil, problem(option, "duplicate ability choice")
		}
		seen[ability] = true
		local := ctx
		// A fixed branch keeps the choice out of mutable game state.
		local.chosenAbility = &ability
		body, err := compileEffect(continuation, local)
		if err != nil {
			return nil, err
		}
		plans[i] = func(bindings instanceBindings) mage.Effect {
			effects := make([]mage.Effect, len(body))
			for j, build := range body {
				effects[j] = build(bindings)
			}
			return mage.Sequence(effects...)
		}
		labels[i] = ability.label()
	}
	return buildResolutionChoice(plans, labels), nil
}

func buildResolutionChoice(plans []effectFactory, labels []string) effectFactory {
	return func(bindings instanceBindings) mage.Effect {
		options := make([]mage.ResolutionOption, len(plans))
		for i, build := range plans {
			body := build(bindings)
			label := body.Text()
			if labels != nil {
				label = labels[i]
			}
			options[i] = mage.ResolutionOption{Label: label, Body: body}
		}
		return mage.ChooseAtResolution(mage.You, options...)
	}
}
