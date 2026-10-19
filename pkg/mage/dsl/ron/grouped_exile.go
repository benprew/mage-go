package ron

import (
	"fmt"
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type groupedExilePlan struct {
	name     string
	singular []bool
	source   int
}

func compileGroupedExile(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	subjects, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	for _, subject := range subjects {
		if !savedGraveyardExileSubject(subject) {
			continue
		}
		first, err := compileSavedGraveyardExile(n, ctx)
		if err != nil {
			return nil, err
		}
		body, err := compileActionSequence(remaining, ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.Effect {
			effects := make([]mage.Effect, 0, 1+len(body))
			effects = append(effects, first(bindings))
			for _, build := range body {
				effects = append(effects, build(bindings))
			}
			return mage.Sequence(effects...)
		}, nil
	}
	if len(subjects) < 2 {
		return nil, problem(n, "grouped exile requires at least two subject groups")
	}
	modifiers, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(modifiers) != 0 {
		return nil, unsupported(n.Args[1], "grouped exile modifiers")
	}
	plan := &groupedExilePlan{name: "exile " + n.Position.String(), singular: make([]bool, len(subjects)), source: -1}
	sets := make([]permanentSetFactory, len(subjects))
	for i, subject := range subjects {
		if err := arity(subject, 1); err != nil {
			return nil, err
		}
		switch subject.Name {
		case "Permanent":
			ref, err := compilePermanentRef(subject.Args[0], ctx)
			if err != nil {
				return nil, err
			}
			if death := ctx.delayedDeath; death != nil && (ref.kind == permanentDeath && ref.death == death || death.watched != nil && ref == *death.watched) {
				return nil, problem(subject, "delayed death graveyard exile requires one subject, not permanent groups")
			}
			plan.singular[i] = true
			if ref.kind == permanentSource {
				if plan.source >= 0 {
					plan.source = -2
				} else if plan.source == -1 {
					plan.source = i
				}
			}
			sets[i] = func(bindings instanceBindings) mage.PermanentSet { return mage.SingletonPermanent(ref.build(bindings)) }
		case "EachPermanent":
			sets[i], err = compileContextualCollection(subject.Args[0], ctx)
			if err != nil {
				return nil, err
			}
		default:
			return nil, unsupported(subject, "grouped exile subject")
		}
	}
	ctx.groupedExile = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.groupedExiles = maps.Clone(bindings.groupedExiles)
		if bindings.groupedExiles == nil {
			bindings.groupedExiles = make(map[*groupedExilePlan][]*mage.CardsBinding)
		}
		results := make([]*mage.CardsBinding, len(sets))
		groups := make([]mage.PermanentExileGroup, len(sets))
		for i, build := range sets {
			results[i] = mage.BindCards(fmt.Sprintf("%s group %d", plan.name, i))
			groups[i] = mage.PermanentExileGroup{Subjects: build(bindings), Result: results[i]}
		}
		bindings.groupedExiles[plan] = results
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.ExilePermanentGroupsInto(groups...))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}

func groupedExileIndex(n Node, ctx scope) (int, error) {
	if n.Kind != Symbol || ctx.groupedExile == nil {
		return 0, problem(n, "exile result requires a bound grouped producer and a symbol")
	}
	plan := ctx.groupedExile
	if n.Name == "ThisExiledCard" && plan.source >= 0 {
		return plan.source, nil
	}
	if len(plan.singular) == 2 {
		switch n.Name {
		case "TheFirstCardExiledThisWay":
			if plan.singular[0] && plan.singular[1] {
				return 0, nil
			}
		case "TheSecondCardExiledThisWay":
			if plan.singular[0] && plan.singular[1] {
				return 1, nil
			}
		case "TheSinglePermanentExiledThisWay":
			if plan.singular[0] != plan.singular[1] {
				if plan.singular[0] {
					return 0, nil
				}
				return 1, nil
			}
		case "TheOtherPermanentsExiledThisWay":
			if plan.singular[0] != plan.singular[1] {
				if plan.singular[0] {
					return 1, nil
				}
				return 0, nil
			}
		}
	}
	return 0, problem(n, "ambiguous or incompatible grouped exile result")
}

func compileGroupedExileGraveyard(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if ctx.sourceZone != nil && ctx.sourceZone.zone == core.ZoneExile && n.Args[0].Name == ctx.sourceZone.name {
		card, err := compileCardRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.Effect {
			return mage.PutExiledCardsInOwnersGraveyards(mage.ExiledCardFrom(card.build(bindings)))
		}, nil
	}
	index, err := groupedExileIndex(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if !ctx.groupedExile.singular[index] {
		return nil, problem(n.Args[0], "single exiled card action requires a singular group")
	}
	plan := ctx.groupedExile
	return func(bindings instanceBindings) mage.Effect {
		return mage.PutExiledCardsInOwnersGraveyards(mage.ExiledCardsFrom(bindings.groupedExiles[plan][index]))
	}, nil
}
