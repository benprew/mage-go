package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileAttachmentMove(n Node, ctx scope) (effectFactory, error) {
	return compileAttachmentMoveFor(n, ctx, ctx.player, false)
}

func compileOptionalAttachment(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if n.Args[1].Kind != Tuple || n.Args[1].Name != "AttachPermanentToAPermanent" {
		return nil, unsupported(n.Args[1], "optional player action")
	}
	return compileAttachmentMoveFor(n.Args[1], ctx, player, true)
}

func compileAttachmentMoveFor(n Node, ctx scope, player playerReferencePlan, optional bool) (effectFactory, error) {
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
		if optional {
			return mage.MayAttachPermanentToChoice(player.build(bindings), attachment.build(bindings), hosts(bindings))
		}
		return mage.AttachPermanentToChoice(player.build(bindings), attachment.build(bindings), hosts(bindings))
	}, nil
}
