package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileStaticAttackerController(n Node, ctx scope) (mage.PlayerRef, error) {
	if n.Name != "ControlledByPlayer" {
		return nil, unsupported(n, "static attacking controller count")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	player := n.Args[0]
	if player.Name != "PlayerOrControllerOfPlaneswalkerCreatureIsAttacking" {
		return nil, unsupported(player, "static attacking player reference")
	}
	if err := arity(player, 1); err != nil {
		return nil, err
	}
	if !ctx.permanent || ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(player, "attacking player count requires a static permanent context")
	}
	if player.Args[0].Kind != Symbol || player.Args[0].Name != "ThisPermanent" {
		return nil, problem(player.Args[0], "static attacking player count requires the source permanent")
	}
	attacker, err := compileConditionalSubject(player.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return mage.DefendingPlayerOf(attacker), nil
}

func compileStaticCountCollection(n Node, ctx scope) (permanentSetFactory, error) {
	if n.Name == "ControlledByPlayer" {
		player, err := compileStaticAttackerController(n, ctx)
		if err != nil {
			return nil, err
		}
		return func(instanceBindings) mage.PermanentSet {
			return mage.PermanentsControlledBy(player, mage.AnyPermanent)
		}, nil
	}
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(n, "static count filter requires operands")
		}
		plans := make([]permanentSetFactory, len(parts))
		for i, part := range parts {
			plans[i], err = compileStaticCountCollection(part, ctx)
			if err != nil {
				return nil, err
			}
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			sets := make([]mage.PermanentSet, len(plans))
			for i, plan := range plans {
				sets[i] = plan(bindings)
			}
			return mage.IntersectPermanents(sets...)
		}, nil
	}
	if err := validateStaticCountFilter(n); err != nil {
		return nil, err
	}
	return compileContextualCollection(n, ctx)
}
