package ron

import "github.com/benprew/mage-go/pkg/mage/core"

func compileXSpendingColor(n Node, hasX bool) (*core.Color, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	value, err := color(n.Args[0])
	if err != nil {
		return nil, err
	}
	if value < core.White || value > core.Green {
		return nil, problem(n.Args[0], "supply one mana color")
	}
	if !hasX {
		return nil, problem(n, "X spending requires a printed X mana cost")
	}
	return &value, nil
}
