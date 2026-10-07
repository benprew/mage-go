package ron

import "github.com/benprew/mage-go/pkg/mage"

type activatedGrantPlan struct {
	costs []costFactory
	body  actionPlan
}

func compileActivatedGrant(n Node) (activatedGrantPlan, error) {
	if n.Name != "Activated" {
		return activatedGrantPlan{}, unsupported(n, "granted activated ability")
	}
	if err := arity(n, 2); err != nil {
		return activatedGrantPlan{}, err
	}
	costs, hasX, err := compileCosts(n.Args[0])
	if err != nil {
		return activatedGrantPlan{}, err
	}
	if hasX {
		return activatedGrantPlan{}, unsupported(n.Args[0], "granted X cost")
	}
	if n.Args[1].Name != "ActionList" {
		return activatedGrantPlan{}, unsupported(n.Args[1], "untargeted granted body")
	}
	// The recipient cannot read the donor's bindings.
	body, err := compileActions(n.Args[1], scope{permanent: true, activation: true})
	if err != nil {
		return activatedGrantPlan{}, err
	}
	if len(body.effects) == 0 {
		return activatedGrantPlan{}, problem(n.Args[1], "granted body requires at least one action")
	}
	if len(body.targets) != 0 {
		return activatedGrantPlan{}, unsupported(n.Args[1], "untargeted granted body")
	}
	return activatedGrantPlan{costs: costs, body: body}, nil
}

func (p activatedGrantPlan) build() (mage.CostGroup, mage.ResolvingEffect) {
	costs := make([]mage.Cost, len(p.costs))
	for i, build := range p.costs {
		costs[i] = build()
	}
	return mage.Costs(costs...), mage.Sequence(p.body.build().effects...)
}

func compileActivatedGrantModification(n Node, subject permanentReferencePlan) (func(instanceBindings) mage.ContinuousModification, error) {
	ability, err := singleActivatedGrant(n)
	if err != nil {
		return nil, err
	}
	plan, err := compileActivatedGrant(ability)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.ContinuousModification {
		costs, body := plan.build()
		return mage.ActivatedAbilityModification(subject.build(bindings), costs, body)
	}, nil
}

func singleActivatedGrant(n Node) (Node, error) {
	if err := arity(n, 1); err != nil {
		return Node{}, err
	}
	abilities, err := list(n.Args[0])
	if err != nil {
		return Node{}, err
	}
	if len(abilities) != 1 {
		return Node{}, problem(n, "supply one granted activated ability")
	}
	return abilities[0], nil
}

func compileStaticActivatedGrant(n Node, parts []Node, ctx scope) (optionFactory, error) {
	if len(parts) != 1 || parts[0].Name != "AddAbility" {
		return nil, unsupported(n, "combined activated ability grant")
	}
	ability, err := singleActivatedGrant(parts[0])
	if err != nil {
		return nil, err
	}
	plan, err := compileActivatedGrant(ability)
	if err != nil {
		return nil, err
	}
	if n.Name == "EachPermanentLayerEffect" {
		subjects, err := compileStaticPermanentFilter(n.Args[0])
		if err != nil {
			return nil, err
		}
		return func() mage.CardOption {
			costs, body := plan.build()
			return mage.WithAbility(mage.Static(mage.GrantEachActivatedAbility(subjects.build(), costs, body)))
		}, nil
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if subject.kind != permanentSource && subject.kind != permanentHost {
		return nil, unsupported(n.Args[0], "static grant subject")
	}
	return func() mage.CardOption {
		costs, body := plan.build()
		return mage.WithAbility(mage.Static(mage.GrantActivatedAbility(subject.build(instanceBindings{}), costs, body)))
	}, nil
}
