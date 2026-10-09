package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileObjectTargetBody(n Node, ctx scope) (optionFactory, error) {
	if legacy, err := compileTargetedColor(n, ctx); err == nil {
		return legacy, nil
	}
	if ctx.permanent || ctx.castingCondition != nil || ctx.castingCost != nil {
		return nil, unsupported(n, "spell/permanent target context")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	branches, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(branches) != 2 {
		return nil, problem(n, "supply one spell branch and one permanent branch")
	}
	var spellBody, permanentBody actionPlan
	var spellPlan *spellTargetPlan
	var permanentPlan *permanentTargetPlan
	for _, branch := range branches {
		if branch.Name != "Targeted" {
			return nil, unsupported(branch, "spell/permanent target branch")
		}
		if err := arity(branch, 2); err != nil {
			return nil, err
		}
		targets, err := list(branch.Args[0])
		if err != nil {
			return nil, err
		}
		if len(targets) != 1 {
			return nil, problem(branch, "supply one target per branch")
		}
		target := targets[0]
		if err := arity(target, 1); err != nil {
			return nil, err
		}
		child := ctx
		child.spellTarget = nil
		child.permanentTarget = nil
		switch target.Name {
		case "TargetSpell":
			if spellPlan != nil {
				return nil, problem(target, "duplicate target category")
			}
			if target.Args[0].Kind != Symbol || target.Args[0].Name != "AnySpell" {
				return nil, unsupported(target.Args[0], "union spell filter")
			}
			spellPlan = &spellTargetPlan{name: "Ref_TargetSpell"}
			child.spellTarget = spellPlan
			spellBody, err = compileActions(branch.Args[1], child)
		case "TargetPermanent":
			if permanentPlan != nil {
				return nil, problem(target, "duplicate target category")
			}
			if target.Args[0].Kind != Symbol || target.Args[0].Name != "AnyPermanent" {
				return nil, unsupported(target.Args[0], "union permanent filter")
			}
			permanentPlan = &permanentTargetPlan{name: "Ref_TargetPermanent"}
			child.permanentTarget = permanentPlan
			permanentBody, err = compileActions(branch.Args[1], child)
		default:
			return nil, unsupported(target, "union target")
		}
		if err != nil {
			return nil, err
		}
	}
	if len(spellBody.targets) != 0 || len(permanentBody.targets) != 0 {
		return nil, problem(n, "nested target declarations are not supported")
	}
	return func() mage.CardOption {
		target := mage.DeclareSpellOrPermanentTarget("spell or permanent")
		spell := buildObjectTargetBranch(spellBody, target, spellPlan, nil)
		permanent := buildObjectTargetBranch(permanentBody, target, nil, permanentPlan)
		return mage.WithAbility(mage.Spell(mage.ResolveObjectTarget(target, spell, permanent), mage.WithTargets(target)))
	}, nil
}

func buildObjectTargetBranch(plan actionPlan, target *mage.SpellOrPermanentTargetBinding, spell *spellTargetPlan, permanent *permanentTargetPlan) mage.ResolvingEffect {
	bindings := initializeActionBindings(instanceBindings{})
	if spell != nil {
		bindings.spells[spell] = mage.SpellFromObjectTarget(target)
	}
	if permanent != nil {
		bindings.permanents[permanent] = mage.PermanentFromObjectTarget(target)
	}
	effects := make([]mage.Effect, len(plan.effects))
	for i, build := range plan.effects {
		effects[i] = build(bindings)
	}
	return mage.Sequence(effects...)
}
