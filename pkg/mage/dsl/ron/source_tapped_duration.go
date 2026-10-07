package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileTemporaryDuration(n Node, ctx scope) (func([]mage.ContinuousModification) mage.Effect, error) {
	if n.Name == "UntilNextUpkeep" || n.Name == "UntilEndOfNextUpkeep" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		if n.Args[0].Kind != Symbol || n.Args[0].Name != "You" {
			return nil, unsupported(n.Args[0], "next upkeep player")
		}
		return func(values []mage.ContinuousModification) mage.Effect {
			if n.Name == "UntilEndOfNextUpkeep" {
				return mage.UntilEndOfNextUpkeep(mage.You, values...)
			}
			return mage.UntilNextUpkeep(mage.You, values...)
		}, nil
	}
	if n.Kind == Symbol && n.Name == "UntilEndOfTurn" {
		return func(values []mage.ContinuousModification) mage.Effect { return mage.Until(core.EndOfTurn, values...) }, nil
	}
	if (ctx.sourceAttack || (ctx.activation && !ctx.delayedStep)) && n.Kind == Symbol && n.Name == "UntilEndOfCombat" {
		return func(values []mage.ContinuousModification) mage.Effect { return mage.Until(core.EndOfCombat, values...) }, nil
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
