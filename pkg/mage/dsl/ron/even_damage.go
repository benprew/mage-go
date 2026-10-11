package ron

import "github.com/benprew/mage-go/pkg/mage"

type damageTargetsPlan struct{ name string }

func compilePluralUnionDamage(n, target Node, ctx scope) (actionPlan, error) {
	var plan actionPlan
	if target.Kind != Symbol {
		return plan, problem(target, "AnyNumberOfAnyTargets requires a symbol")
	}
	if !ctx.spellTargets || ctx.permanent || ctx.activation || ctx.delayedStep || ctx.drawReplacementBody || ctx.targetNamespace != "" || ctx.damageTargets != nil {
		return plan, unsupported(target, "plural union targets outside an ordinary spell")
	}
	binding := &damageTargetsPlan{name: "Ref_AnyTargets"}
	ctx.damageTargets = binding
	var err error
	plan, err = compileActions(n.Args[1], ctx)
	if err != nil {
		return plan, err
	}
	if len(plan.targets) != 0 {
		return plan, problem(n, "nested target declarations are not supported")
	}
	plan.targets = []targetFactory{func(instanceBindings) mage.Target { return mage.DeclareAnyNumberDamageTargets(binding.name) }}
	plan.damageTargets = binding
	return plan, nil
}

func compileEvenSpellDamage(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	if !ctx.spellTargets || ctx.permanent || ctx.activation || ctx.delayedStep || ctx.drawReplacementBody || ctx.targetNamespace != "" {
		return nil, unsupported(n, "equal damage outside an ordinary spell")
	}
	source, err := compileSelfDamageSource(n.Args[0], ctx, false)
	if err != nil {
		return nil, err
	}
	total, err := compileInstanceValue(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	recipients := n.Args[2]
	if recipients.Name == "Ref_AnyTargets" {
		if recipients.Kind != Symbol || ctx.damageTargets == nil {
			return nil, problem(recipients, "unbound plural union target reference")
		}
		binding := ctx.damageTargets
		return func(bindings instanceBindings) mage.Effect {
			return mage.DealDamageEvenly(source(bindings), bindings.recipientSets[binding], total(bindings))
		}, nil
	}
	if recipients.Name != "EachPermanent" {
		return nil, unsupported(recipients, "equal damage recipient collection")
	}
	subjects, err := compilePermanentDamageCollection(recipients, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.DealDamageEvenlyToRecipients(source(bindings), mage.PermanentDamageRecipients(subjects(bindings)), total(bindings))
	}, nil
}
