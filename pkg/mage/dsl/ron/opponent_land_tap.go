package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileOpponentLandTap(n Node) (func(mage.Effect) *mage.GenericTriggered, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	filter := n.Args[0]
	if filter.Name != "And" {
		return nil, unsupported(filter, "opponent land tap filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	parts, err := list(filter.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 2 {
		return nil, problem(filter, "supply one land subtype and one opponent controller filter")
	}
	var subtype string
	var opponent bool
	for _, part := range parts {
		if err := arity(part, 1); err != nil {
			return nil, err
		}
		switch part.Name {
		case "IsLandType":
			if subtype != "" {
				return nil, problem(part, "duplicate land subtype filter")
			}
			name, err := symbol(part.Args[0])
			if err != nil {
				return nil, err
			}
			subtype = subtypeName(name)
			if err := mage.ValidateSubtypeFamily(mage.SubtypeLand, subtype); err != nil {
				return nil, problem(part.Args[0], err.Error())
			}
			// Family checks allow new names. The event requires a known land subtype.
			pattern := mage.OpponentLandBecomesTapped().OfSubtype(subtype)
			declaration := mage.NewEnchantment("Land tap declaration", "", mage.WithAbility(mage.WhenEvent(pattern, mage.GainLifeFor(mage.You, mage.Fixed(0)))))
			if err := mage.ValidateCardDefinition(declaration.Name(), declaration); err != nil {
				return nil, problem(part.Args[0], err.Error())
			}
		case "ControlledByAPlayer":
			if opponent {
				return nil, problem(part, "duplicate opponent controller filter")
			}
			if part.Args[0].Kind != Symbol || part.Args[0].Name != "Opponent" {
				return nil, unsupported(part.Args[0], "land tap controller filter")
			}
			opponent = true
		default:
			return nil, unsupported(part, "opponent land tap filter")
		}
	}
	if subtype == "" || !opponent {
		return nil, problem(filter, "supply one land subtype and one opponent controller filter")
	}
	return func(effect mage.Effect) *mage.GenericTriggered {
		return mage.WhenEvent(mage.OpponentLandBecomesTapped().OfSubtype(subtype), effect).(*mage.GenericTriggered).WithLandTapTextRevision()
	}, nil
}
