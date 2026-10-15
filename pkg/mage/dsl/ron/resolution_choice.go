package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCheckableAbilityChoice(n, continuation Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "keyword removal choice requires an ordinary resolving body")
	}
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
		return nil, unsupported(continuation, "keyword removal choice continuation")
	}
	if err := arity(continuation, 3); err != nil {
		return nil, err
	}
	if continuation.Args[2].Kind != Symbol || continuation.Args[2].Name != "UntilEndOfTurn" {
		return nil, unsupported(continuation.Args[2], "keyword removal choice duration")
	}
	modifications, err := list(continuation.Args[1])
	if err != nil {
		return nil, err
	}
	if len(modifications) != 1 || modifications[0].Name != "LosesAbility" {
		return nil, problem(continuation, "keyword removal choice requires one LosesAbility continuation")
	}
	removal := modifications[0]
	if err := arity(removal, 1); err != nil {
		return nil, err
	}
	if removal.Args[0].Kind != Symbol || removal.Args[0].Name != "TheChosenAbility" {
		return nil, problem(removal, "keyword removal choice must use TheChosenAbility")
	}
	plans := make([]effectFactory, len(options))
	labels := make([]string, len(options))
	seen := map[string]bool{}
	for i, option := range options {
		keyword, err := compileTemporaryKeyword(option)
		if err != nil {
			return nil, err
		}
		label := keyword.String()
		if seen[label] {
			return nil, problem(option, "duplicate ability choice")
		}
		seen[label] = true
		// Each branch has a fixed keyword, so copies cannot share a saved choice.
		branch := continuation
		branch.Args = append([]Node(nil), continuation.Args...)
		removal.Args = []Node{option}
		branch.Args[1].Args = []Node{removal}
		plans[i], err = compileTemporaryLayer(branch, ctx)
		if err != nil {
			return nil, err
		}
		labels[i] = label
	}
	return buildResolutionChoiceFor(plans, labels, ctx.player), nil
}

func compileActionChoice(n Node, ctx scope) (effectFactory, error) {
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
		if len(parts) == 0 {
			return nil, problem(option, "action choice requires a nonempty body")
		}
		body, err := compileActionSequence(parts, ctx)
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
	}
	return buildResolutionChoiceFor(plans, nil, ctx.player), nil
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
	return buildResolutionChoiceFor(plans, labels, ctx.player), nil
}

func buildResolutionChoice(plans []effectFactory, labels []string) effectFactory {
	return buildResolutionChoiceFor(plans, labels, playerReferencePlan{})
}

func buildResolutionChoiceFor(plans []effectFactory, labels []string, player playerReferencePlan) effectFactory {
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
		return mage.ChooseAtResolution(player.build(bindings), options...)
	}
}
