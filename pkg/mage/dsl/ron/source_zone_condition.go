package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileSourceZoneCondition(n Node, ctx scope) (func(instanceBindings) mage.StateCondition, error) {
	count := 4
	if n.Name == "ExiledCardPassesFilter" {
		count = 2
	}
	if err := arity(n, count); err != nil {
		return nil, err
	}
	if ctx.sourceZone == nil {
		return nil, problem(n, "condition requires a source zone card")
	}
	card, err := compileCardRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if card.sourceZone != ctx.sourceZone {
		return nil, problem(n.Args[0], "condition requires the source zone card")
	}
	filterNode := n.Args[count-1]
	if n.Name == "ExiledCardPassesFilter" {
		if ctx.sourceZone.zone != core.ZoneExile {
			return nil, problem(n, "exile condition requires an exile source")
		}
		if filterNode.Name == "HasACounterOfType" || filterNode.Name == "HasNoCountersOfType" {
			if err := arity(filterNode, 1); err != nil {
				return nil, err
			}
			counter, err := compilePermanentCounterType(filterNode.Args[0])
			if err != nil {
				return nil, err
			}
			return func(bindings instanceBindings) mage.StateCondition {
				if filterNode.Name == "HasNoCountersOfType" {
					return mage.ExiledCardHasNoCounters(card.build(bindings), counter)
				}
				return mage.ExiledCardHasCounter(card.build(bindings), counter)
			}, nil
		}
	}
	if filterNode.Name != "IsCardtype" {
		return nil, unsupported(filterNode, "source-zone card filter")
	}
	if err := arity(filterNode, 1); err != nil {
		return nil, err
	}
	kind, err := cardType(filterNode.Args[0])
	if err != nil {
		return nil, err
	}
	filter := mage.NewCardFilter(kind.String()+" card", func(c mage.Card) bool { return c.HasType(kind) })
	switch n.Name {
	case "ExiledCardPassesFilter":
		if ctx.sourceZone.zone != core.ZoneExile {
			return nil, problem(n, "exile condition requires an exile source")
		}
		return func(bindings instanceBindings) mage.StateCondition {
			return mage.CardMatchesFilter(card.build(bindings), filter)
		}, nil
	case "CardIsInPlayersGraveyardWithANumberOfCardsAboveIt":
		if ctx.sourceZone.zone != core.ZoneGraveyard {
			return nil, problem(n, "ordered condition requires a graveyard source")
		}
		if n.Args[1].Kind != Symbol || n.Args[1].Name != "You" {
			return nil, unsupported(n.Args[1], "source graveyard player")
		}
		comparison := n.Args[2]
		if comparison.Name != "GreaterThanOrEqualTo" {
			return nil, unsupported(comparison, "ordered graveyard comparison")
		}
		if err := arity(comparison, 1); err != nil {
			return nil, err
		}
		value := comparison.Args[0]
		if value.Name != "Integer" {
			return nil, unsupported(value, "ordered graveyard count")
		}
		if err := arity(value, 1); err != nil {
			return nil, err
		}
		if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
			return nil, problem(value, "ordered graveyard count requires a nonnegative integer")
		}
		minimum := value.Args[0].Integer
		return func(bindings instanceBindings) mage.StateCondition {
			return mage.GraveyardCardHasAtLeastAbove(card.build(bindings), mage.You, minimum, filter)
		}, nil
	default:
		return nil, unsupported(n, "source-zone condition")
	}
}
