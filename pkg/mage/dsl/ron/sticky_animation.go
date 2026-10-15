package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileStickyAnimation(n Node) (optionFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	filter := n.Args[0]
	if filter.Kind != Tuple || filter.Name != "And" || len(filter.Args) != 1 || filter.Args[0].Kind != List || len(filter.Args[0].Args) != 2 {
		return nil, problem(filter, "sticky animation requires noncreature artifacts")
	}
	seen := map[string]bool{}
	for _, part := range filter.Args[0].Args {
		if part.Kind != Tuple || len(part.Args) != 1 || part.Args[0].Kind != Symbol || seen[part.Name] || ((part.Name != "IsNonCardtype" || part.Args[0].Name != "Creature") && (part.Name != "IsCardtype" || part.Args[0].Name != "Artifact")) {
			return nil, problem(part, "sticky animation requires noncreature artifacts")
		}
		seen[part.Name] = true
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) != 4 {
		return nil, problem(n.Args[1], "supply ability removal, artifact and creature types, and mana-value base P/T")
	}
	seen = map[string]bool{}
	for _, part := range parts {
		key := part.Name
		switch part.Name {
		case "LosesAllAbilities":
			if part.Kind != Symbol {
				return nil, problem(part, "ability removal requires a symbol")
			}
		case "AddCardtype":
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			kind := part.Args[0]
			if kind.Kind != Symbol || kind.Name != "Artifact" && kind.Name != "Creature" {
				return nil, unsupported(kind, "sticky animation type")
			}
			key += kind.Name
		case "SetPowerAndToughnessBoth":
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			value := part.Args[0]
			if value.Kind != Tuple || value.Name != "ManaValueOfPermanent" || len(value.Args) != 1 || value.Args[0].Kind != Symbol || value.Args[0].Name != "EachablePermanent" {
				return nil, problem(value, "sticky animation requires each member's mana value")
			}
		default:
			return nil, unsupported(part, "sticky animation modification")
		}
		if seen[key] {
			return nil, problem(part, "duplicate sticky animation modification")
		}
		seen[key] = true
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "sticky animation departure duration")
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.Static(mage.AnimateArtifactsUntilCleanupAfterExit()))
	}, nil
}
