package ron

import "github.com/benprew/mage-go/pkg/mage"

func validatePluralSacrificeResult(n Node, ctx scope) error {
	if n.Kind != Symbol || n.Name != "ThePermanentsSacrificedThisWay" || ctx.pluralSacrifice == nil || ctx.enteringSource || ctx.drawReplacementBody {
		return problem(n, "unbound or incompatible plural sacrifice payment result")
	}
	return nil
}

func savedGraveyardExileSubject(n Node) bool {
	return len(n.Args) == 1 && ((n.Name == "EachCardInGraveyards" && n.Args[0].Name == "ThePermanentsSacrificedThisWay") || (n.Name == "CardInGraveyards" && n.Args[0].Name == "ThisPermanent"))
}

func compileSavedGraveyardExile(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	subjects, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(subjects) == 0 || len(operations) != 0 {
		return nil, problem(n, "saved graveyard exile requires subjects and no modifiers")
	}
	builds := make([]func(instanceBindings) mage.CardSet, len(subjects))
	for i, subject := range subjects {
		if err := arity(subject, 1); err != nil {
			return nil, err
		}
		switch subject.Name {
		case "EachCardInGraveyards":
			if err := validatePluralSacrificeResult(subject.Args[0], ctx); err != nil {
				return nil, err
			}
			builds[i] = func(bindings instanceBindings) mage.CardSet {
				return mage.FromCards(bindings.pluralSacrifices[ctx.pluralSacrifice].cards)
			}
		case "CardInGraveyards":
			ref := subject.Args[0]
			if ref.Kind != Symbol || ref.Name != "ThisPermanent" || ctx.sourceCardPayment == nil || ctx.sourceCardPayment.producers != 1 {
				return nil, problem(ref, "saved source graveyard card requires one source payment")
			}
			ctx.sourceCardPayment.used = true
			builds[i] = func(bindings instanceBindings) mage.CardSet {
				return mage.SingletonCard(mage.FromCard(bindings.sourceCards[ctx.sourceCardPayment]))
			}
		default:
			return nil, unsupported(subject, "saved graveyard exile subject")
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		sets := make([]mage.CardSet, len(builds))
		for i, build := range builds {
			sets[i] = build(bindings)
		}
		return mage.ExileGraveyardCards(sets...)
	}, nil
}
