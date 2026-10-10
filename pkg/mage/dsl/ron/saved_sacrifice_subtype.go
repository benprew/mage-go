package ron

import "github.com/benprew/mage-go/pkg/mage"

type sacrificeSubtypePlan struct {
	label   string
	subtype string
	cost    *Node
	filter  mage.PermanentFilter
}

func sacrificeSubtypeBinding(bindings instanceBindings, plan *sacrificeSubtypePlan) *mage.NumberBinding {
	if saved := bindings.sacrificeSubtypes[plan]; saved != nil {
		return saved
	}
	saved := mage.BindNumber("saved sacrifice subtype " + plan.label)
	bindings.sacrificeSubtypes[plan] = saved
	return saved
}

func activationSacrificeSubtypePlan(cost Node) *sacrificeSubtypePlan {
	var producers []Node
	var scan func(Node)
	scan = func(n Node) {
		if n.Name == "And" && len(n.Args) == 1 && n.Args[0].Kind == List {
			for _, part := range n.Args[0].Args {
				scan(part)
			}
		} else if n.Name == "SacrificeAPermanent" || n.Name == "SacrificePermanent" {
			producers = append(producers, n)
		}
	}
	scan(cost)
	if len(producers) != 1 || producers[0].Name != "SacrificeAPermanent" {
		return nil
	}
	return &sacrificeSubtypePlan{cost: &producers[0], label: producers[0].Position.String()}
}

func compileSacrificeSubtypeBranch(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	condition := n.Args[0]
	if err := arity(condition, 1); err != nil {
		return nil, err
	}
	if ctx.sacrificeSubtype == nil {
		return nil, problem(condition, "saved sacrifice property requires a prior local sacrifice producer")
	}
	filter := condition.Args[0]
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	var family mage.SubtypeFamily
	switch filter.Name {
	case "IsLandType":
		if _, err := compileLandDependenceFilter(filter); err != nil {
			return nil, err
		}
		family = mage.SubtypeLand
	case "IsCreatureType":
		family = mage.SubtypeCreature
	default:
		return nil, unsupported(filter, "saved sacrifice subtype filter")
	}
	subtype, err := symbol(filter.Args[0])
	if err != nil {
		return nil, err
	}
	subtype = subtypeName(subtype)
	if err := mage.ValidateSubtypeFamily(family, subtype); err != nil {
		return nil, problem(filter, err.Error())
	}
	result := ctx.sacrificeSubtype
	if result.subtype != "" && result.subtype != subtype {
		return nil, problem(condition, "one sacrifice producer supports one saved subtype")
	}
	result.subtype = subtype
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(n, "saved sacrifice branch requires actions")
	}
	effects, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		saved := sacrificeSubtypeBinding(bindings, result)
		body := make([]mage.Effect, len(effects))
		for i, build := range effects {
			body[i] = build(bindings)
		}
		return mage.IfPositive(mage.FromNumber(saved), mage.Sequence(body...))
	}, nil
}
