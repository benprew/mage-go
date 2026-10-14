package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileLiveStaticCollection(n Node, ctx scope) (mage.PermanentSet, error) {
	if !ctx.permanent || ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "live static collection requires a battlefield source declaration")
	}
	switch n.Name {
	case "SinglePermanent", "BandedWith", "Other":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		if n.Args[0].Kind != Symbol || n.Args[0].Name != "ThisPermanent" {
			return nil, unsupported(n.Args[0], "live static collection source")
		}
		if n.Name == "BandedWith" {
			return mage.BandedWith(mage.This), nil
		}
		if n.Name == "Other" {
			return mage.WithoutPermanents(mage.MatchingPermanents(mage.AnyPermanent), mage.SingletonPermanent(mage.This)), nil
		}
		return mage.SingletonPermanent(mage.This), nil
	case "ControlledByAPlayer":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		player := n.Args[0]
		if player.Name != "SinglePlayer" {
			return nil, unsupported(player, "live static collection player")
		}
		if err := arity(player, 1); err != nil {
			return nil, err
		}
		if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
			return nil, unsupported(player.Args[0], "live static collection controller")
		}
		return mage.PermanentsControlledBy(mage.You, mage.AnyPermanent), nil
	case "And", "Or":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(n, "live static collection must not be empty")
		}
		sets := make([]mage.PermanentSet, len(parts))
		for i, part := range parts {
			sets[i], err = compileLiveStaticCollection(part, ctx)
			if err != nil {
				return nil, err
			}
		}
		if n.Name == "Or" {
			return mage.UnionPermanents(sets...), nil
		}
		return mage.IntersectPermanents(sets...), nil
	default:
		filter, err := compileStaticConditionFilter(n)
		if err != nil {
			return nil, err
		}
		return mage.MatchingPermanents(filter), nil
	}
}

func hasLiveStaticCollection(n Node) bool {
	if n.Name == "Other" || n.Name == "SinglePermanent" || n.Name == "BandedWith" || n.Name == "ControlledByAPlayer" {
		return true
	}
	return slices.ContainsFunc(n.Args, hasLiveStaticCollection)
}

func compileStaticCollectionPrevention(n Node, ctx scope, condition mage.StateCondition) (optionFactory, error) {
	event := n.Args[0]
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	from, err := compileFilter(event.Args[0])
	if err != nil {
		return nil, err
	}
	recipient := event.Args[1]
	if recipient.Name != "APermanent" {
		return nil, unsupported(recipient, "live static prevention recipient")
	}
	if err := arity(recipient, 1); err != nil {
		return nil, err
	}
	subjects, err := compileLiveStaticCollection(recipient.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Kind != Symbol || operations[0].Name != "PreventThatDamage" {
		return nil, problem(n.Args[1], "live static prevention requires PreventThatDamage")
	}
	if condition == nil {
		condition = mage.StateConditionFunc("live static prevention", []mage.StateDependency{mage.StateBattlefield}, func(mage.StateReader, mage.StateContext) bool { return true })
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.Static(mage.PreventDamageFromMatchingPermanentsToSetIf(subjects, from, condition)))
	}, nil
}
