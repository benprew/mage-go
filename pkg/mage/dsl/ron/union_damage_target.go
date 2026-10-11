package ron

import (
	"strconv"

	"github.com/benprew/mage-go/pkg/mage"
)

func isUnionDamageTarget(n Node) bool {
	return n.Name == "AnyTarget" || n.Name == "AnyTargetOfAPlayersChoice"
}

func compileUnionDamageTarget(n Node, ctx scope, name string) (targetFactory, error) {
	if n.Name == "AnyTarget" {
		if n.Kind != Symbol {
			return nil, problem(n, "AnyTarget requires a symbol")
		}
	} else {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		if n.Args[0].Kind != Symbol || n.Args[0].Name != "Opponent" {
			return nil, unsupported(n.Args[0], "union target chooser")
		}
	}
	return func(instanceBindings) mage.Target {
		if n.Name == "AnyTargetOfAPlayersChoice" {
			return mage.DeclareOpponentChosenDamageTarget(ctx.targetNamespace + name)
		}
		return mage.DeclareDamageTarget(ctx.targetNamespace + name)
	}, nil
}

func compileUnionDamageGroups(n Node, targets []Node, ctx scope) (actionPlan, error) {
	var plan actionPlan
	ctx.damageTarget = nil
	ctx.damageGroups = make(map[string]*damageTargetPlan, len(targets))
	groups := make([]*damageTargetPlan, len(targets))
	builds := make([]targetFactory, len(targets))
	for i, target := range targets {
		if !isUnionDamageTarget(target) {
			return plan, unsupported(target, "singular union target group")
		}
		name := "Ref_AnyTarget" + strconv.Itoa(i+1)
		build, err := compileUnionDamageTarget(target, ctx, name)
		if err != nil {
			return plan, err
		}
		groups[i] = &damageTargetPlan{name: name}
		ctx.damageGroups[name] = groups[i]
		builds[i] = build
	}
	plan, err := compileActions(n.Args[1], ctx)
	if err != nil {
		return plan, err
	}
	if len(plan.targets) != 0 {
		return plan, problem(n, "nested target declarations are not supported")
	}
	plan.targets = builds
	plan.damageGroups = groups
	return plan, nil
}
