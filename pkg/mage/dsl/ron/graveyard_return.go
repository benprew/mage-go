package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type cardTargetPlan struct {
	name         string
	creatureOnly bool
	ante         bool
}

type deathCardPlan struct{ name string }

type cardReferencePlan struct {
	sourceZone *sourceZonePlan
	exile      *groupedExilePlan
	exileGroup int
	target     *cardTargetPlan
	death      *deathCardPlan
}

func compileCardRef(n Node, ctx scope) (cardReferencePlan, error) {
	if n.Kind == Symbol && ctx.sourceZone != nil && n.Name == ctx.sourceZone.name {
		return cardReferencePlan{sourceZone: ctx.sourceZone}, nil
	}
	switch n.Name {
	case "TheFirstCardExiledThisWay", "TheSecondCardExiledThisWay", "TheSinglePermanentExiledThisWay", "ThisExiledCard":
		index, err := groupedExileIndex(n, ctx)
		if err != nil {
			return cardReferencePlan{}, err
		}
		if !ctx.groupedExile.singular[index] {
			return cardReferencePlan{}, problem(n, "card reference requires a singular exile group")
		}
		return cardReferencePlan{exile: ctx.groupedExile, exileGroup: index}, nil
	}

	if n.Kind == Symbol && ctx.deathCard != nil && n.Name == ctx.deathCard.name {
		return cardReferencePlan{death: ctx.deathCard}, nil
	}
	if n.Kind == Symbol && ctx.cardTarget != nil && n.Name == ctx.cardTarget.name {
		return cardReferencePlan{target: ctx.cardTarget}, nil
	}
	return cardReferencePlan{}, problem(n, "unbound or incompatible card reference")
}

func (p cardReferencePlan) build(bindings instanceBindings) mage.CardRef {
	if p.sourceZone != nil {
		return mage.FromCard(bindings.sourceZones[p.sourceZone])
	}
	if p.exile != nil {
		return mage.SingleExiledCardFrom(bindings.groupedExiles[p.exile][p.exileGroup])
	}

	if p.death != nil {
		return mage.FromCard(bindings.deathCards[p.death])
	}
	return bindings.cards[p.target]
}

func compileGraveyardTarget(n Node, ctx scope) (targetFactory, string, error) {
	filters, own, err := compileGraveyardTargetFilter(n.Args[0])
	if err != nil {
		return nil, "", err
	}
	return func(instanceBindings) mage.Target {
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
	if card.exile != nil || (card.sourceZone != nil && card.sourceZone.zone != core.ZoneGraveyard) || (card.target != nil && card.target.ante) {
		return nil, problem(n.Args[0], "graveyard return requires a graveyard card reference")
	}
	if count == 1 {
		return func(bindings instanceBindings) mage.Effect { return mage.ReturnCardToHand(card.build(bindings)) }, nil
	}
	if card.target == nil || !card.target.creatureOnly {
		return nil, problem(n, "battlefield return requires a creature card target")
	}
	entries, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(entries) != 1 {
		return nil, unsupported(n.Args[1], "graveyard battlefield entry")
	}
	var controller playerReferencePlan
	entry := entries[0]
	switch entry.Name {
	case "EntersNormally":
		if _, err := symbol(entry); err != nil {
			return nil, err
		}
	case "EntersUnderPlayersControl":
		if err := arity(entry, 1); err != nil {
			return nil, err
		}
		controller, err = compilePlayerRef(entry.Args[0], ctx)
		if err != nil {
			return nil, err
		}
	default:
		return nil, unsupported(entry, "graveyard battlefield entry")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.ReturnCardToBattlefield(card.build(bindings), controller.build(bindings))
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
