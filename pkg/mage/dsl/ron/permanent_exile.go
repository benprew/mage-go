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
	if subject.Name == "EachCardInGraveyards" {
		return compileGraveyardBatchExile(subject, n.Args[1], ctx)
	}
	if subject.Name == "Spell" {
		return compileSpellExile(subject, n.Args[1], ctx)
	}
	if subject.Name == "TheTopCardOfLibrary" {
		if subject.Kind != Symbol || !ctx.permanent {
			return nil, problem(subject, "library-top linked exile requires a permanent source and a symbol")
		}
		modifiers, err := list(n.Args[1])
		if err != nil {
			return nil, err
		}
		faceDown := false
		if len(modifiers) != 0 {
			if len(modifiers) != 1 || modifiers[0].Kind != Symbol || modifiers[0].Name != "ExilesFaceDown" {
				return nil, unsupported(n.Args[1], "library-top exile modifiers")
			}
			faceDown = true
		}
		return func(bindings instanceBindings) mage.Effect {
			return mage.ExileLibraryTopLinked(ctx.player.build(bindings), producerExileLink(ctx), faceDown)
		}, nil
	}
	if subject.Name == "CardInGraveyards" {
		if err := arity(subject, 1); err != nil {
			return nil, err
		}
		card, err := compileCardRef(subject.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		if card.exile != nil || (card.target != nil && card.target.ante) {
			return nil, problem(subject.Args[0], "graveyard exile requires a graveyard card reference")
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
	return func(bindings instanceBindings) mage.Effect {
		if ctx.permanent {
			return mage.ExilePermanentLinked(ref.build(bindings), producerExileLink(ctx))
		}
		return mage.Exile(ref.build(bindings))
	}, nil
}

func producerExileLink(ctx scope) *mage.ExileLink {
	if ctx.exileLink != nil {
		return ctx.exileLink
	}
	return mage.NewExileLink()
}
