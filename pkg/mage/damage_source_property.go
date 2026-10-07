package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type damageSourceFilterKind uint8

const (
	damageSourceAny damageSourceFilterKind = iota
	damageSourceColor
	damageSourceType
	damageSourceAll
	damageSourceEither
)

// DamageSourceFilter reads the information supplied by a typed source reference.
// The zero value permits any valid source.
type DamageSourceFilter struct {
	kind     damageSourceFilterKind
	color    core.Color
	cardType core.CardType
	operands []DamageSourceFilter
}

// DamageSourceHasColor requires one of the five Magic colors.
func DamageSourceHasColor(color core.Color) DamageSourceFilter {
	return DamageSourceFilter{kind: damageSourceColor, color: color}
}

// DamageSourceHasType requires the specified card type.
func DamageSourceHasType(cardType core.CardType) DamageSourceFilter {
	return DamageSourceFilter{kind: damageSourceType, cardType: cardType}
}

// AllDamageSources requires each supplied source property.
func AllDamageSources(filters ...DamageSourceFilter) DamageSourceFilter {
	return DamageSourceFilter{kind: damageSourceAll, operands: slices.Clone(filters)}
}

// AnyOfDamageSources requires at least one supplied source property.
func AnyOfDamageSources(filters ...DamageSourceFilter) DamageSourceFilter {
	return DamageSourceFilter{kind: damageSourceEither, operands: slices.Clone(filters)}
}

// Match reads source information without a game query or state change.
// A saved source supplies saved information; a live reference supplies live information.
func (f DamageSourceFilter) Match(source DamageSource) bool {
	if !source.Valid() {
		return false
	}
	switch f.kind {
	case damageSourceAny:
		return true
	case damageSourceColor:
		return source.HasColor(f.color)
	case damageSourceType:
		return source.HasType(f.cardType)
	case damageSourceAll:
		return len(f.operands) > 0 && !slices.ContainsFunc(f.operands, func(part DamageSourceFilter) bool { return !part.Match(source) })
	case damageSourceEither:
		return slices.ContainsFunc(f.operands, func(part DamageSourceFilter) bool { return part.Match(source) })
	default:
		return false
	}
}

func (f DamageSourceFilter) inspectDefinition() definitionNode {
	n := definitionNode{Role: "damage-source-filter"}
	switch f.kind {
	case damageSourceAny:
		n.Text = "any source"
	case damageSourceColor:
		n.Text = "source color"
		if f.color < core.White || f.color > core.Green {
			n.problem = "supply one of the five Magic colors"
		}
	case damageSourceType:
		n.Text = "source card type"
		if f.cardType < core.TypeCreature || f.cardType > core.TypePlaneswalker {
			n.problem = "supply a valid card type"
		}
	case damageSourceAll, damageSourceEither:
		n.Text = "source property composition"
		if len(f.operands) == 0 {
			n.problem = "supply at least one source filter"
		}
		for i, part := range f.operands {
			n.children = append(n.children, definitionEdge{fmt.Sprintf("filters[%d]", i), part})
		}
	default:
		n.problem = "invalid damage source filter"
	}
	return n
}
