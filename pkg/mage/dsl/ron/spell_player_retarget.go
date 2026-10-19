package ron

import "github.com/benprew/mage-go/pkg/mage"

func compilePlayerSpellRetarget(n Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || !ctx.activation || ctx.delayedStep || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, unsupported(n, "resolving spell retarget context")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	spell, err := compileSpellRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if n.Args[1].Kind != Symbol || n.Args[1].Name != "AnyPlayer" {
		return nil, unsupported(n.Args[1], "retarget player filter")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.ChangeSingleSpellTargetToPlayer(spell.build(bindings))
	}, nil
}

func compileSpellPlayerBranch(n Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || !ctx.activation || ctx.delayedStep || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, unsupported(n, "resolving spell condition context")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	condition := n.Args[0]
	if err := arity(condition, 2); err != nil {
		return nil, err
	}
	spell, err := compileSpellRef(condition.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	filter := condition.Args[1]
	if filter.Name != "TargetsPlayer" {
		return nil, unsupported(filter, "spell player condition")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(filter.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(n.Args[1], "condition requires actions")
	}
	if ctx.activation && containsManaAction(n) {
		return nil, problem(n, "conditional mana activation requires classification and solver planning")
	}
	body, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		effects := make([]mage.Effect, len(body))
		for i, build := range body {
			effects[i] = build(bindings)
		}
		return mage.IfSpellTargetsPlayer(spell.build(bindings), player.build(bindings), mage.Sequence(effects...))
	}, nil
}
