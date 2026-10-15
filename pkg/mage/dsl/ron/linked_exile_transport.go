package ron

import (
	"maps"
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileExileAssociation(rules []Node, permanent bool) (*mage.ExileLink, error) {
	producers := 0
	var consumer *Node
	var walk func(Node)
	walk = func(n Node) {
		if n.Name == "WasExiledWithPermanent" {
			declaration := n
			consumer = &declaration
		}
		if n.Name == "Exile" && n.Kind == Tuple && len(n.Args) == 2 && n.Args[0].Kind == List && len(n.Args[0].Args) == 1 {
			subject := n.Args[0].Args[0]
			if subject.Name == "TheTopCardOfLibrary" || subject.Name == "Permanent" {
				producers++
			}
		}
		for _, arg := range n.Args {
			walk(arg)
		}
		for _, key := range slices.Sorted(maps.Keys(n.Fields)) {
			walk(n.Fields[key])
		}
	}
	for _, rule := range rules {
		walk(rule)
	}
	if consumer == nil {
		return nil, nil
	}
	if !permanent || producers != 1 {
		return nil, problem(*consumer, "linked exile requires one unambiguous permanent-source producer")
	}
	return mage.NewExileLink(), nil
}

func compileLinkedExileTransport(n Node, ctx scope) (effectFactory, error) {
	args := 1
	if n.Name == "PutEachExiledCardOntoTheBattlefield" {
		args = 2
	}
	if err := arity(n, args); err != nil {
		return nil, err
	}
	var host *permanentReferencePlan
	var counters *counterSnapshotPlan
	if args == 2 {
		options := n.Args[1]
		if options.Kind != List || len(options.Args) < 1 || len(options.Args) > 2 || options.Args[0].Kind != Symbol || options.Args[0].Name != "EntersUnderOwnersControl" {
			return nil, problem(options, "supply owner control and at most one attachment or noted-counter option")
		}
		if len(options.Args) == 2 {
			option := options.Args[1]
			if option.Name == "EntersWithNotedCounters" {
				if option.Kind != Symbol || ctx.counterSnapshot == nil || !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
					return nil, problem(option, "noted counters require a bound resolving snapshot and a symbol")
				}
				counters = ctx.counterSnapshot
			} else {
				ref, err := compileAttachedEntry(option, ctx)
				if err != nil {
					return nil, err
				}
				host = &ref
			}
		}
	}
	entryOptions := func(bindings instanceBindings) []mage.BattlefieldEntryOption {
		if counters != nil {
			return []mage.BattlefieldEntryOption{mage.EnterWithSavedCounters(bindings.counterSnapshots[counters])}
		}
		if host == nil {
			return nil
		}
		return []mage.BattlefieldEntryOption{mage.EnterAttachedTo(host.build(bindings))}
	}
	set := n.Args[0]
	if set.Kind == Symbol && set.Name == "TheOtherPermanentsExiledThisWay" {
		index, err := groupedExileIndex(set, ctx)
		if err != nil {
			return nil, err
		}
		plan := ctx.groupedExile
		return func(bindings instanceBindings) mage.Effect {
			subjects := mage.ExiledCardsFrom(bindings.groupedExiles[plan][index])
			if n.Name == "ReturnEachExiledCardToOwnersHand" {
				return mage.ReturnExiledCardsToOwnersHands(subjects)
			}
			if n.Name == "PutEachExiledCardOntoTheBattlefield" {
				return mage.PutExiledCardsOntoBattlefieldUnderOwnersControl(subjects, entryOptions(bindings)...)
			}
			return mage.PutExiledCardsInOwnersGraveyards(subjects)
		}, nil
	}
	if set.Name != "WasExiledWithPermanent" {
		return nil, unsupported(set, "exiled card set")
	}
	if err := arity(set, 1); err != nil {
		return nil, err
	}
	if ctx.exileLink == nil {
		return nil, problem(set, "linked exile requires a producer declaration")
	}
	ref, err := compilePermanentRef(set.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentSource {
		return nil, problem(set.Args[0], "linked exile consumer requires its source permanent")
	}
	return func(bindings instanceBindings) mage.Effect {
		subjects := mage.CardsExiledWith(ctx.exileLink, ref.build(bindings))
		if n.Name == "ReturnEachExiledCardToOwnersHand" {
			return mage.ReturnExiledCardsToOwnersHands(subjects)
		}
		if n.Name == "PutEachExiledCardOntoTheBattlefield" {
			return mage.PutExiledCardsOntoBattlefieldUnderOwnersControl(subjects, entryOptions(bindings)...)
		}
		return mage.PutExiledCardsInOwnersGraveyards(subjects)
	}, nil
}
