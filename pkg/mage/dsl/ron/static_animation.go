package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileStaticAnimation(n Node, parts []Node, subjects staticPermanentFilterPlan) (optionFactory, error) {
	var power, toughness int
	var colors []core.Color
	var hasPT, hasType, hasColors bool
	for _, part := range parts {
		switch part.Name {
		case "SetPT":
			if hasPT {
				return nil, problem(part, "duplicate animation base values")
			}
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			pt := part.Args[0]
			if pt.Name != "PT" {
				return nil, unsupported(pt, "static animation base values")
			}
			if err := arity(pt, 2); err != nil {
				return nil, err
			}
			if pt.Args[0].Kind != IntegerNode || pt.Args[1].Kind != IntegerNode {
				return nil, problem(pt, "static animation requires integer base values")
			}
			power, toughness = pt.Args[0].Integer, pt.Args[1].Integer
			hasPT = true
		case "AddCardtype":
			if hasType {
				return nil, problem(part, "duplicate animation type")
			}
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			if part.Args[0].Kind != Symbol || part.Args[0].Name != "Creature" {
				return nil, unsupported(part.Args[0], "static animation type")
			}
			hasType = true
		case "SetColor":
			if hasColors {
				return nil, problem(part, "duplicate animation color list")
			}
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			value := part.Args[0]
			if value.Name != "SimpleColorList" {
				return nil, unsupported(value, "static animation colors")
			}
			if err := arity(value, 1); err != nil {
				return nil, err
			}
			nodes, err := list(value.Args[0])
			if err != nil {
				return nil, err
			}
			colors = make([]core.Color, 0, len(nodes))
			for _, node := range nodes {
				c, err := color(node)
				if err != nil {
					return nil, err
				}
				if c == core.Colorless {
					return nil, problem(node, "use an empty color list for colorless")
				}
				colors = append(colors, c)
			}
			hasColors = true
		default:
			return nil, unsupported(part, "static animation modification")
		}
	}
	if !hasPT || !hasType {
		return nil, problem(n, "static animation requires base values and creature type")
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.Static(mage.AnimateEach(subjects.build(), power, toughness, colors...)))
	}, nil
}
