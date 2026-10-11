package ron

import "github.com/benprew/mage-go/pkg/mage"

type distributedDamagePlan struct {
	name  string
	total int
}

func compileDistributedDamage(n Node, ctx scope) (actionPlan, error) {
	var plan actionPlan
	if err := arity(n, 3); err != nil {
		return plan, err
	}
	if !ctx.spellTargets || ctx.permanent || ctx.activation || ctx.targetNamespace != "" {
		return plan, unsupported(n, "distributed damage outside an ordinary spell")
	}
	targets, err := list(n.Args[0])
	if err != nil {
		return plan, err
	}
	if len(targets) != 1 || targets[0].Kind != Symbol || targets[0].Name != "AnyNumberOfAnyTargets" {
		return plan, problem(n.Args[0], "distributed damage requires one any-number union target group")
	}
	division := n.Args[1]
	if division.Name != "DistributeNumberAmongTargets" {
		return plan, unsupported(division, "target distribution")
	}
	if err := arity(division, 1); err != nil {
		return plan, err
	}
	value := division.Args[0]
	if value.Name != "Integer" {
		return plan, unsupported(value, "fixed distribution total")
	}
	if err := arity(value, 1); err != nil {
		return plan, err
	}
	if value.Args[0].Kind != IntegerNode || value.Args[0].Integer <= 0 {
		return plan, problem(value, "distribution total must be a positive integer")
	}
	body := n.Args[2]
	if body.Name != "ActionList" {
		return plan, unsupported(body, "distributed damage body")
	}
	if err := arity(body, 1); err != nil {
		return plan, err
	}
	actions, err := list(body.Args[0])
	if err != nil {
		return plan, err
	}
	if len(actions) != 1 || actions[0].Name != "SpellDealsDistributedDamage" {
		return plan, problem(body, "distribution requires one spell damage instruction")
	}
	if err := arity(actions[0], 1); err != nil {
		return plan, err
	}
	source := actions[0].Args[0]
	if source.Kind != Symbol || source.Name != "ThisSpell" {
		return plan, unsupported(source, "distributed spell damage source")
	}
	plan.distribution = &distributedDamagePlan{name: "Ref_AnyTargets", total: value.Args[0].Integer}
	return plan, nil
}

func (p *distributedDamagePlan) build() Actions {
	recipients := mage.DeclareAnyNumberDamageTargets(p.name)
	effect := mage.DealAllocatedDamage(mage.This, recipients, p.total)
	spell := mage.Spell(effect, mage.WithTargets(recipients))
	return Targeted(spell.Targets(), ActionList(effect))
}
