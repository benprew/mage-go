package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileExactAttachment(n Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.drawReplacementBody {
		return nil, unsupported(n, "nonresolving exact attachment")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	attachment, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	host, err := compilePermanentRef(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.AttachToPermanent(attachment.build(bindings), host.build(bindings))
	}, nil
}
