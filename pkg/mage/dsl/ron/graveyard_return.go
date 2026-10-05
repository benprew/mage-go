package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
)

type cardTargetPlan struct {
	name         string
	creatureOnly bool
}

type cardReferencePlan struct{ target *cardTargetPlan }

func compileCardRef(n Node, ctx scope) (cardReferencePlan, error) {
	if n.Kind == Symbol && ctx.cardTarget != nil && n.Name == ctx.cardTarget.name {
		return cardReferencePlan{target: ctx.cardTarget}, nil
	}
	return cardReferencePlan{}, problem(n, "unbound or incompatible card reference")
}

func (p cardReferencePlan) build(bindings instanceBindings) mage.CardRef {
	return bindings.cards[p.target]
}

func compileGraveyardTarget(n Node, ctx scope) (targetFactory, string, error) {
	filters, own, err := compileGraveyardTargetFilter(n.Args[0])
	if err != nil {
		return nil, "", err
	}
	return func() mage.Target {
		options := make([]mage.GraveyardTargetOption, 0, len(filters)+1)
		for _, filter := range filters {
			options = append(options, mage.GraveyardCardFilter(filter))
		}
		if !own {
			options = append(options, mage.InAnyGraveyard())
		}
		return mage.DeclareGraveyardCardTarget(ctx.targetNamespace+"Ref_TargetGraveyardCard", options...)
	}, "Ref_TargetGraveyardCard", nil
}

func compileGraveyardTargetFilter(n Node) ([]mage.CardFilter, bool, error) {
	if err := arity(n, 1); err != nil {
		return nil, false, err
	}
	switch n.Name {
	case "And":
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, false, err
		}
		if len(parts) == 0 {
			return nil, false, problem(n, "graveyard filter requires at least one operand")
		}
		var filters []mage.CardFilter
		own := false
		for _, part := range parts {
			compiled, restricted, err := compileGraveyardTargetFilter(part)
			if err != nil {
				return nil, false, err
			}
			filters = append(filters, compiled...)
			own = own || restricted
		}
		return filters, own, nil
	case "IsCardtype":
		kind, err := cardType(n.Args[0])
		if err != nil {
			return nil, false, err
		}
		filter := mage.NewCardFilter(kind.String()+" card", func(card mage.Card) bool { return card.HasType(kind) })
		return []mage.CardFilter{filter}, false, nil
	case "InAPlayersGraveyard":
		players := n.Args[0]
		if players.Kind == Symbol && players.Name == "AnyPlayer" {
			return nil, false, nil
		}
		if players.Name != "SinglePlayer" {
			return nil, false, unsupported(players, "graveyard player filter")
		}
		if err := arity(players, 1); err != nil {
			return nil, false, err
		}
		if players.Args[0].Kind != Symbol || players.Args[0].Name != "You" {
			return nil, false, unsupported(players.Args[0], "graveyard player reference")
		}
		return nil, true, nil
	default:
		return nil, false, unsupported(n, "graveyard card filter")
	}
}

func compileGraveyardReturn(n Node, ctx scope) (effectFactory, error) {
	count := 1
	if n.Name == "PutGraveyardCardOntoBattlefield" {
		count = 2
	}
	if err := arity(n, count); err != nil {
		return nil, err
	}
	card, err := compileCardRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if count == 1 {
		return func(bindings instanceBindings) mage.Effect { return mage.ReturnCardToHand(card.build(bindings)) }, nil
	}
	if !card.target.creatureOnly {
		return nil, problem(n, "battlefield return requires a creature card target")
	}
	entries, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(entries) != 1 || entries[0].Kind != Symbol || entries[0].Name != "EntersNormally" {
		return nil, unsupported(n.Args[1], "graveyard battlefield entry")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.ReturnCardToBattlefield(card.build(bindings), mage.You)
	}, nil
}

func graveyardFilterRequiresCreature(n Node) bool {
	if n.Name == "IsCardtype" {
		return n.Args[0].Name == "Creature"
	}
	if n.Name == "And" {
		if slices.ContainsFunc(n.Args[0].Args, graveyardFilterRequiresCreature) {
			return true
		}
	}
	return false
}
