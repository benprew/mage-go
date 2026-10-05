package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileOptionalTapChoice(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	action := n.Args[0]
	if action.Name != "TapOrUntapPermanent" {
		return nil, unsupported(action, "optional action")
	}
	if err := arity(action, 1); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(action.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.MayTapOrUntap(subject.build(bindings))
	}, nil
}
