package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileAttachmentMove(n Node, ctx scope) (effectFactory, error) {
	if ctx.enteringSource || ctx.drawReplacementBody {
		return nil, unsupported(n, "nonresolving attachment movement")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	attachment, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	hosts, err := compileContextualCollection(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.AttachPermanentToChoice(ctx.player.build(bindings), attachment.build(bindings), hosts(bindings))
	}, nil
}
