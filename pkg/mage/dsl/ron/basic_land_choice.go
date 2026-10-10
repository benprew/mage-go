package ron

import "github.com/benprew/mage-go/pkg/mage"

type basicLandEntryChoicePlan struct{}

func compileBasicLandEntryDeclaration(rules []Node, permanent bool) (*basicLandEntryChoicePlan, error) {
	var plan *basicLandEntryChoicePlan
	for _, rule := range rules {
		if rule.Name != "AsPermanentEnters" || len(rule.Args) != 2 || rule.Args[1].Kind != List {
			continue
		}
		for _, operation := range rule.Args[1].Args {
			if operation.Name != "ChooseABasicLandType" {
				continue
			}
			if !permanent {
				return nil, unsupported(rule, "land type entry choice on a nonpermanent")
			}
			if _, err := symbol(operation); err != nil {
				return nil, err
			}
			if len(rule.Args[1].Args) != 1 || rule.Args[0].Kind != Symbol || rule.Args[0].Name != "ThisPermanent" {
				return nil, problem(rule, "land type entry requires one source choice")
			}
			if plan != nil {
				return nil, problem(rule, "duplicate basic land type entry choice")
			}
			plan = &basicLandEntryChoicePlan{}
		}
	}
	return plan, nil
}

func compileChosenLandTypeLayer(n Node, parts []Node, ctx scope) (optionFactory, error) {
	if n.Name != "PermanentLayerEffect" || n.Args[0].Kind != Symbol || n.Args[0].Name != "HostPermanent" || !ctx.aura || ctx.basicLandEntryChoice == nil {
		return nil, problem(n, "chosen land type requires an Aura host and an entry declaration")
	}
	if len(parts) != 1 {
		return nil, unsupported(n.Args[1], "combined chosen land type modifications")
	}
	if err := validateChosenLandType(parts[0]); err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.StaticAbility(mage.BecomesChosenBasicLandTypeAttachedEffect()).WithLayeredLandTypeTextRevision())
	}, nil
}

func validateChosenLandType(n Node) error {
	if err := arity(n, 1); err != nil {
		return err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "TheChosenLandType" {
		return problem(n.Args[0], "unbound or incompatible basic land type choice")
	}
	return nil
}

func compileBasicLandChoice(choice, continuation Node, ctx scope) (effectFactory, error) {
	if _, err := symbol(choice); err != nil {
		return nil, err
	}
	if continuation.Name != "CreatePermanentLayerEffectUntil" {
		return nil, unsupported(continuation, "basic land choice continuation")
	}
	if err := arity(continuation, 3); err != nil {
		return nil, err
	}
	parts, err := list(continuation.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) != 1 || parts[0].Name != "AddAbility" {
		return nil, problem(continuation, "land type choice requires one landwalk grant")
	}
	if err := arity(parts[0], 1); err != nil {
		return nil, err
	}
	abilities, err := list(parts[0].Args[0])
	if err != nil {
		return nil, err
	}
	if len(abilities) != 1 || abilities[0].Name != "Landwalk" {
		return nil, problem(parts[0], "land type choice requires one landwalk grant")
	}
	landwalk := abilities[0]
	if err := arity(landwalk, 1); err != nil {
		return nil, err
	}
	filter := landwalk.Args[0]
	if filter.Name != "IsLandTypeVariable" {
		return nil, unsupported(filter, "chosen landwalk filter")
	}
	if err := validateChosenLandType(filter); err != nil {
		return nil, err
	}
	labels := []string{"Plains", "Island", "Swamp", "Mountain", "Forest"}
	plans := make([]effectFactory, len(labels))
	for i, landType := range labels {
		// Fixed branches keep the choice inside one resolution.
		fixed := filter
		fixed.Name = "IsLandType"
		fixed.Args = []Node{{Kind: Symbol, Name: landType, Position: filter.Args[0].Position}}
		fixedWalk := landwalk
		fixedWalk.Args = []Node{fixed}
		grant := parts[0]
		grant.Args = []Node{{Kind: List, Args: []Node{fixedWalk}, Position: parts[0].Args[0].Position}}
		body := continuation
		body.Args = []Node{continuation.Args[0], {Kind: List, Args: []Node{grant}, Position: continuation.Args[1].Position}, continuation.Args[2]}
		effects, err := compileEffect(body, ctx)
		if err != nil {
			return nil, err
		}
		plans[i] = func(bindings instanceBindings) mage.Effect {
			values := make([]mage.Effect, len(effects))
			for j, build := range effects {
				values[j] = build(bindings)
			}
			return mage.Sequence(values...)
		}
	}
	return buildResolutionChoice(plans, labels), nil
}
