package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileAttachedEntry(n Node, ctx scope) (permanentReferencePlan, error) {
	if err := arity(n, 1); err != nil {
		return permanentReferencePlan{}, err
	}
	if n.Name != "EntersAttachedToPermanent" || !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return permanentReferencePlan{}, unsupported(n, "explicit entry attachment")
	}
	return compilePermanentRef(n.Args[0], ctx)
}

func compileDepartedAuraAttachedEntry(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if ctx.hostDeath == nil || !ctx.aura || !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, unsupported(n, "departed Aura entry context")
	}
	entries, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(entries) != 2 || entries[0].Name != "EntersUnderPlayersControl" {
		return nil, unsupported(n.Args[0], "departed Aura attached entry")
	}
	if err := arity(entries[0], 1); err != nil {
		return nil, err
	}
	controller, err := compilePlayerRef(entries[0].Args[0], ctx)
	if err != nil {
		return nil, err
	}
	host, err := compileAttachedEntry(entries[1], ctx)
	if err != nil {
		return nil, err
	}
	ctx.hostDeath.departedAura = true
	return func(bindings instanceBindings) mage.Effect {
		return mage.ReturnCardToBattlefield(mage.FromCard(bindings.hostDeaths[ctx.hostDeath].aura), controller.build(bindings), mage.EnterAttachedTo(host.build(bindings)))
	}, nil
}
