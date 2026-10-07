package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func isPaidUntapChoice(n Node) bool {
	return n.Name == "PlayerAction" && len(n.Args) >= 2 && n.Args[1].Name == "ChooseAnyNumberPermanentsAndPayManaForEach"
}

func compilePaidUntap(choice, continuation Node, ctx scope) (effectFactory, error) {
	if err := arity(choice, 2); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(choice.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	action := choice.Args[1]
	if err := arity(action, 2); err != nil {
		return nil, err
	}
	filter, err := compilePaidUntapFilter(action.Args[0])
	if err != nil {
		return nil, err
	}
	text, err := manaCost(action.Args[1])
	if err != nil {
		return nil, err
	}
	cost := core.ParseManaCost(text)
	if cost.HasX {
		return nil, problem(action.Args[1], "selected-permanent payment requires a fixed mana cost")
	}
	if continuation.Name != "PlayerAction" {
		return nil, problem(continuation, "paid choice requires an adjacent untap of its result")
	}
	if err := arity(continuation, 2); err != nil {
		return nil, err
	}
	_, err = compilePlayerRef(continuation.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if definitionKey(choice.Args[0]) != definitionKey(continuation.Args[0]) {
		return nil, problem(continuation.Args[0], "paid choice and untap must use the same player")
	}
	untap := continuation.Args[1]
	if untap.Name != "UntapEachPermanent" {
		return nil, unsupported(untap, "paid-choice continuation")
	}
	if err := arity(untap, 1); err != nil {
		return nil, err
	}
	if untap.Args[0].Kind != Symbol || untap.Args[0].Name != "TheChosenPermanents" {
		return nil, unsupported(untap.Args[0], "paid-choice result")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.UntapWithManaPayment(player.build(bindings), filter, cost)
	}, nil
}

func compilePaidUntapFilter(n Node) (mage.PermanentFilter, error) {
	if n.Name != "And" {
		return mage.PermanentFilter{}, unsupported(n, "paid-choice filter")
	}
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return mage.PermanentFilter{}, err
	}
	if len(parts) != 2 {
		return mage.PermanentFilter{}, problem(n, "supply a permanent filter and one action-player controller")
	}
	var filter mage.PermanentFilter
	bound := false
	for _, part := range parts {
		if part.Name == "ControlledByPlayer" {
			if err := arity(part, 1); err != nil {
				return mage.PermanentFilter{}, err
			}
			if bound || part.Args[0].Kind != Symbol || part.Args[0].Name != "PlayerAction_ThatPlayer" {
				return mage.PermanentFilter{}, problem(part, "controller must be the current action player")
			}
			bound = true
		} else {
			if !filter.IsZero() {
				return mage.PermanentFilter{}, problem(part, "supply one permanent filter")
			}
			if !paidUntapRequiresTapped(part) {
				return mage.PermanentFilter{}, problem(part, "paid untap requires a tapped-permanent filter")
			}
			filter, err = compileFilter(part)
			if err != nil {
				return mage.PermanentFilter{}, err
			}
		}
	}
	if !bound || filter.IsZero() {
		return mage.PermanentFilter{}, problem(n, "supply a permanent filter and one action-player controller")
	}
	return filter, nil
}

func paidUntapRequiresTapped(n Node) bool {
	if n.Kind == Symbol {
		return n.Name == "IsTapped"
	}
	if n.Name != "And" || len(n.Args) != 1 || n.Args[0].Kind != List {
		return false
	}
	return slices.ContainsFunc(n.Args[0].Args, paidUntapRequiresTapped)
}
