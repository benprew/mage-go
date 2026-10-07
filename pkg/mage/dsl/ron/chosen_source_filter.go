package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileChosenDamageSourceFilter(n Node) (mage.DamageSourceFilter, error) {
	switch n.Name {
	case "AnyDamageSource":
		if _, err := symbol(n); err != nil {
			return mage.DamageSourceFilter{}, err
		}
		return mage.DamageSourceFilter{}, nil
	case "IsColor":
		if err := arity(n, 1); err != nil {
			return mage.DamageSourceFilter{}, err
		}
		value, err := color(n.Args[0])
		if err != nil {
			return mage.DamageSourceFilter{}, err
		}
		if value == core.Colorless {
			return mage.DamageSourceFilter{}, problem(n.Args[0], "source color requires one of the five Magic colors")
		}
		return mage.DamageSourceHasColor(value), nil
	case "IsCardtype":
		if err := arity(n, 1); err != nil {
			return mage.DamageSourceFilter{}, err
		}
		value, err := cardType(n.Args[0])
		if err != nil {
			return mage.DamageSourceFilter{}, err
		}
		return mage.DamageSourceHasType(value), nil
	case "And", "Or":
		if err := arity(n, 1); err != nil {
			return mage.DamageSourceFilter{}, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.DamageSourceFilter{}, err
		}
		if len(parts) == 0 {
			return mage.DamageSourceFilter{}, problem(n, "supply at least one source filter")
		}
		filters := make([]mage.DamageSourceFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileChosenDamageSourceFilter(part)
			if err != nil {
				return mage.DamageSourceFilter{}, err
			}
		}
		if n.Name == "And" {
			return mage.AllDamageSources(filters...), nil
		}
		return mage.AnyOfDamageSources(filters...), nil
	default:
		return mage.DamageSourceFilter{}, unsupported(n, "chosen damage source filter")
	}
}
