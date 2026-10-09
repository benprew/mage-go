package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileManaTypeReplacement(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "supply one mana type replacement operation")
	}
	op := operations[0]
	event, duration := n.Args[0], n.Args[2]
	switch event.Name {
	case "PlayerTappingAPermanentWouldProduceMana":
		if err := arity(event, 2); err != nil {
			return nil, err
		}
		if event.Args[0].Kind != Symbol || event.Args[0].Name != "You" {
			return nil, unsupported(event.Args[0], "mana replacement player")
		}
		player, err := compilePlayerRef(event.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		filter, err := compileStaticPermanentFilter(event.Args[1])
		if err != nil {
			return nil, err
		}
		if !filter.controlled || filter.excludeSource {
			return nil, problem(event.Args[1], "supply a filter restricted to permanents you control")
		}
		if duration.Kind != Symbol || duration.Name != "UntilEndOfTurn" {
			return nil, unsupported(duration, "mana replacement duration")
		}
		if op.Name != "WouldProduceMana_ReplaceType" {
			return nil, unsupported(op, "mana replacement operation")
		}
		if err := arity(op, 1); err != nil {
			return nil, err
		}
		to, err := compileConcreteManaType(op.Args[0])
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.Effect {
			return mage.ReplaceTappedManaTypeForPlayerThisTurn(player.build(bindings), filter.filter, to)
		}, nil
	case "TappingPermanentWouldProduceMana":
		if err := arity(event, 1); err != nil {
			return nil, err
		}
		subject, err := compilePermanentRef(event.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		if duration.Name != "UntilPermanentLeavesBattlefield" {
			return nil, unsupported(duration, "mana replacement duration")
		}
		if err := arity(duration, 1); err != nil {
			return nil, err
		}
		if !sameDeclaration(event.Args[0], duration.Args[0]) {
			return nil, problem(duration, "mana replacement duration must refer to the same subject")
		}
		if op.Name != "WouldProduceMana_ReplaceTypeOfType" {
			return nil, unsupported(op, "mana replacement operation")
		}
		if err := arity(op, 2); err != nil {
			return nil, err
		}
		from, err := compileConcreteManaType(op.Args[0])
		if err != nil {
			return nil, err
		}
		to, err := compileConcreteManaType(op.Args[1])
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.Effect {
			return mage.ReplaceTappedManaTypeUntilLeaves(subject.build(bindings), from, to)
		}, nil
	default:
		return nil, unsupported(event, "mana replacement event")
	}
}
func compileConcreteManaType(n Node) (core.Color, error) {
	productions, err := compileMana(n)
	if err != nil {
		return core.Colorless, err
	}
	if len(productions) != 1 || n.Kind != Symbol || productions[0].Color == core.AnyColor {
		return core.Colorless, problem(n, "supply one concrete mana type")
	}
	return productions[0].Color, nil
}
