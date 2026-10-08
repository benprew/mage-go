package ron

import "github.com/benprew/mage-go/pkg/mage"

func compilePermanentExile(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	subjects, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(subjects) != 1 {
		return nil, problem(n.Args[0], "exile requires one subject")
	}
	subject := subjects[0]
	if subject.Name == "CardInGraveyards" {
		if err := arity(subject, 1); err != nil {
			return nil, err
		}
		card, err := compileCardRef(subject.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		modifiers, err := list(n.Args[1])
		if err != nil {
			return nil, err
		}
		if len(modifiers) != 0 {
			return nil, unsupported(n.Args[1], "exile modifiers")
		}
		return func(bindings instanceBindings) mage.Effect {
			return mage.ExileGraveyardCard(card.build(bindings))
		}, nil
	}
	if subject.Name == "EachPermanent" {
		if err := arity(subject, 1); err != nil {
			return nil, err
		}
		collection, err := compileContextualCollection(subject.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		modifiers, err := list(n.Args[1])
		if err != nil {
			return nil, err
		}
		if len(modifiers) != 0 {
			return nil, unsupported(n.Args[1], "exile modifiers")
		}
		return func(bindings instanceBindings) mage.Effect { return mage.ExileEach(collection(bindings)) }, nil
	}
	if subject.Name != "Permanent" {
		return nil, unsupported(subject, "exile subject")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(subject.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	modifiers, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(modifiers) != 0 {
		return nil, unsupported(n.Args[1], "exile modifiers")
	}
	return func(bindings instanceBindings) mage.Effect { return mage.Exile(ref.build(bindings)) }, nil
}
