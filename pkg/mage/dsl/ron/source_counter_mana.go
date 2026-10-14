package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func containsCounterMana(n Node) bool {
	if n.Name == "AddManaRepeated" && len(n.Args) == 2 && n.Args[1].Name == "TheTotalNumberOfCountersOfTypeAmongPermanents" {
		return true
	}
	return slices.ContainsFunc(n.Args, containsCounterMana)
}

func compileSourceCounterMana(n Node) (optionFactory, error) {
	if n.Name != "Activated" {
		return nil, unsupported(n, "source-counter mana activation")
	}
	cost := n.Args[0]
	if cost.Name != "TapPermanent" {
		return nil, unsupported(cost, "source-counter mana cost")
	}
	if err := arity(cost, 1); err != nil {
		return nil, err
	}
	if cost.Args[0].Kind != Symbol || cost.Args[0].Name != "ThisPermanent" {
		return nil, unsupported(cost.Args[0], "source-counter mana source")
	}
	body := n.Args[1]
	if body.Name != "ActionList" {
		return nil, unsupported(body, "source-counter mana body")
	}
	if err := arity(body, 1); err != nil {
		return nil, err
	}
	actions, err := list(body.Args[0])
	if err != nil {
		return nil, err
	}
	if len(actions) != 1 || actions[0].Name != "AddManaRepeated" {
		return nil, problem(body, "supply one AddManaRepeated action")
	}
	action := actions[0]
	if err := arity(action, 2); err != nil {
		return nil, err
	}
	if action.Args[0].Kind != Symbol {
		return nil, unsupported(action.Args[0], "source-counter fixed mana type")
	}
	productions, err := compileMana(action.Args[0])
	if err != nil {
		return nil, err
	}
	if len(productions) != 1 || productions[0].Color >= core.AnyColor {
		return nil, unsupported(action.Args[0], "source-counter fixed mana type")
	}
	value := action.Args[1]
	if err := arity(value, 2); err != nil {
		return nil, err
	}
	counter, err := compilePermanentCounterType(value.Args[0])
	if err != nil {
		return nil, err
	}
	subject := value.Args[1]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "source-counter mana subject")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	if subject.Args[0].Kind != Symbol || subject.Args[0].Name != "ThisPermanent" {
		return nil, unsupported(subject.Args[0], "source-counter mana reference")
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.NewSourceCounterManaAbility(productions[0].Color, counter))
	}, nil
}
