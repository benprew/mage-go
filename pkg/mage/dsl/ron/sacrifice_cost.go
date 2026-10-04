package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSacrificeCostFilter(n Node) (mage.PermanentFilter, error) {
	if n.Name != "And" && n.Name != "Or" {
		return compileFilter(n)
	}
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return mage.PermanentFilter{}, err
	}
	if len(parts) == 0 {
		return mage.PermanentFilter{}, problem(n, "sacrifice cost requires a nonempty filter")
	}
	filters := make([]mage.PermanentFilter, len(parts))
	for i, part := range parts {
		filters[i], err = compileSacrificeCostFilter(part)
		if err != nil {
			return mage.PermanentFilter{}, err
		}
	}
	if n.Name == "And" {
		return mage.And(filters...), nil
	}
	return mage.Or(filters...), nil
}
