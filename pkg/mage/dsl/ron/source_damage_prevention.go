package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSourceDamagePrevention(n Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "source damage prevention requires an ordinary resolving body")
	}
	event := n.Args[0]
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	source, err := compilePermanentRef(event.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Kind != Symbol || operations[0].Name != "PreventThatDamage" {
		return nil, problem(n.Args[1], "source damage prevention requires the symbol PreventThatDamage")
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "source damage prevention duration")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.PreventDamageByThisTurn(source.build(bindings))
	}, nil
}
