package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileTemporaryDuration(n Node, ctx scope) (func([]mage.ContinuousModification) mage.Effect, error) {
	if n.Kind == Symbol && n.Name == "UntilEndOfTurn" {
		return func(values []mage.ContinuousModification) mage.Effect { return mage.Until(core.EndOfTurn, values...) }, nil
	}
	if n.Name != "ForAsLongAsPermanentRemainsTapped" {
		return nil, unsupported(n, "duration")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	source, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if source.kind != permanentSource {
		return nil, unsupported(n.Args[0], "tapped duration source")
	}
	return func(values []mage.ContinuousModification) mage.Effect {
		return mage.UntilSourceUntaps(mage.This, values...)
	}, nil
}
