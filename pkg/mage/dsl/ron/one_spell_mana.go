package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileOneSpellMana(n Node, player playerReferencePlan, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, unsupported(n, "one-spell mana context")
	}
	if n.Kind != Tuple || n.Name != "MayActionOnce" {
		return nil, unsupported(n, "one-spell mana permission")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	action := n.Args[0]
	if action.Kind != Tuple || action.Name != "CastASpellAndMaySpendManaAsThoughAnyTypeToPayManaCost" {
		return nil, unsupported(action, "one-spell mana action")
	}
	if err := arity(action, 1); err != nil {
		return nil, err
	}
	if action.Args[0].Kind != Symbol || action.Args[0].Name != "AnySpell" {
		return nil, unsupported(action.Args[0], "one-spell mana filter")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.SpendAnyManaForOneSpellThisTurn(player.build(bindings))
	}, nil
}
