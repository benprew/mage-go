package ron

import "github.com/benprew/mage-go/pkg/mage"

func isDeathCombatCollection(n Node) bool {
	if n.Name != "Or" || n.Kind != Tuple || len(n.Args) != 1 || n.Args[0].Kind != List {
		return false
	}
	for _, part := range n.Args[0].Args {
		if part.Name == "BlockedDeadAttacker" || part.Name == "WasBlockingDeadDefender" {
			return true
		}
	}
	return false
}

func compileDeathCombatCollection(n Node, ctx scope) (permanentSetFactory, error) {
	if !ctx.resolving || !ctx.permanent || ctx.sourceDeathOwner == nil || ctx.permanentEvent == nil || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "death combat collection requires a source-death resolving body")
	}
	parts := n.Args[0].Args
	if len(parts) != 2 || parts[0].Kind != Symbol || parts[1].Kind != Symbol {
		return nil, problem(n, "death combat collection requires both distinct combat relation symbols")
	}
	first, second := parts[0].Name, parts[1].Name
	if first == "WasBlockingDeadDefender" {
		first, second = second, first
	}
	if first != "BlockedDeadAttacker" || second != "WasBlockingDeadDefender" {
		return nil, problem(n, "death combat collection requires both distinct combat relation symbols")
	}
	ctx.permanentEvent.referenced = true
	return func(bindings instanceBindings) mage.PermanentSet {
		return mage.CombatOpponentsWithLKI(mage.From(bindings.tapped[ctx.permanentEvent]))
	}, nil
}
