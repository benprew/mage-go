package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type chosenManaPlan struct{ name string }

func compileChosenManaPayment(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if ctx.enteringSource || ctx.drawReplacementBody {
		return nil, unsupported(n, "chosen resolving mana payment")
	}
	player := ctx.player
	action := n
	if n.Name == "PlayerAction" {
		if err := arity(n, 2); err != nil {
			return nil, err
		}
		var err error
		player, err = compilePlayerRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		action = n.Args[1]
	}
	if _, err := symbol(action); err != nil {
		return nil, err
	}
	plan := &chosenManaPlan{name: "paid mana " + n.Position.String()}
	ctx.chosenMana = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		payer := player.build(bindings)
		bindings.chosenMana = maps.Clone(bindings.chosenMana)
		if bindings.chosenMana == nil {
			bindings.chosenMana = make(map[*chosenManaPlan]*mage.NumberBinding)
		}
		result := mage.BindNumber(plan.name)
		bindings.chosenMana[plan] = result
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.PayChosenManaInto(payer, result))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}

func compileInstructionPrevention(n Node, ctx scope) (effectFactory, error) {
	if ctx.enteringSource || ctx.drawReplacementBody {
		return nil, unsupported(n, "instruction damage prevention")
	}
	if err := arity(n, 4); err != nil {
		return nil, err
	}
	source, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	amount, err := compileInstanceValue(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	recipient, err := compileDamagePacketRecipient(n.Args[2], ctx)
	if err != nil {
		return nil, err
	}
	prevention, err := compileInstanceValue(n.Args[3], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.DealDamageWithPreventionFrom(mage.DamageSourceOf(source.build(bindings)), recipient(bindings), amount(bindings), prevention(bindings))
	}, nil
}
