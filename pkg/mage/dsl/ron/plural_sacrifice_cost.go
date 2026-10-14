package ron

import "github.com/benprew/mage-go/pkg/mage"

type pluralSacrificePlan struct {
	filter mage.PermanentFilter
	source bool
}
type pluralSacrificeBindings struct {
	power *mage.NumberBinding
	cards *mage.CardsBinding
}

func compilePluralSacrificePlan(cost Node) (*pluralSacrificePlan, error) {
	var result *pluralSacrificePlan
	var sources []Node
	var walk func(Node) error
	walk = func(n Node) error {
		if n.Name == "And" {
			if err := arity(n, 1); err != nil {
				return err
			}
			parts, err := list(n.Args[0])
			if err != nil {
				return err
			}
			for _, part := range parts {
				if err := walk(part); err != nil {
					return err
				}
			}
		}
		if n.Name == "SacrificePermanent" {
			sources = append(sources, n)
		}
		if n.Name != "SacrificeAnyNumberOfPermanents" {
			return nil
		}
		if result != nil {
			return problem(n, "supply one plural sacrifice payment producer")
		}
		if err := arity(n, 1); err != nil {
			return err
		}
		hasController := false
		filter, err := compileMatchingTapFilter(n.Args[0], &hasController)
		if err != nil {
			return err
		}
		if !hasController {
			return problem(n, "plural sacrifice payment requires your control restriction")
		}
		result = &pluralSacrificePlan{filter: filter}
		return nil
	}
	if err := walk(cost); err != nil {
		return nil, err
	}
	if result != nil && len(sources) > 0 {
		if len(sources) != 1 {
			return nil, problem(cost, "combined sacrifice requires one source sacrifice")
		}
		source := sources[0]
		if err := arity(source, 1); err != nil {
			return nil, err
		}
		if source.Args[0].Kind != Symbol || source.Args[0].Name != "ThisPermanent" {
			return nil, problem(source, "combined sacrifice requires the source permanent symbol")
		}
		result.source = true
	}
	return result, nil
}
