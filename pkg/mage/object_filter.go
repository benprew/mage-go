package mage

import (
	"slices"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ObjectView supplies characteristics that are common to cards and permanents.
type ObjectView interface {
	Name() string
	HasType(CardType) bool
	HasSuperType(SuperType) bool
	HasSubType(string) bool
	HasKeyword(Keyword) bool
	Colors() []Color
	ManaValue() int
	PowerValue(GameReader) int
	ToughnessValue(GameReader) int
}

func (c *Card) HasKeyword(kw Keyword) bool      { return cardHasKeyword(c, kw) }
func (c *Card) ManaValue() int                  { return c.ManaCost().CMC() }
func (c *Card) PowerValue(_ GameReader) int     { return c.Power() }
func (c *Card) ToughnessValue(_ GameReader) int { return c.Toughness() }
func (p *Permanent) HasSuperType(t SuperType) bool {
	if p.FaceDown {
		return false
	}
	return p.Card.HasSuperType(t)
}
func (p *Permanent) ManaValue() int {
	if p.FaceDown {
		return 0
	}
	return p.Card.ManaValue()
}
func (p *Permanent) PowerValue(g GameReader) int     { return p.CurrentPower(g) }
func (p *Permanent) ToughnessValue(g GameReader) int { return p.CurrentToughness(g) }

// ObjectFilter tests current characteristics in any zone.
// The zero value matches all objects.
type ObjectFilter struct {
	label     string
	predicate func(ObjectView, GameReader) bool
}

// CharacteristicFilter creates a filter that can be used in any zone.
func CharacteristicFilter(label string, predicate func(ObjectView, GameReader) bool) ObjectFilter {
	return ObjectFilter{label: label, predicate: predicate}
}
func (f ObjectFilter) Match(object ObjectView, g GameReader) bool {
	return object != nil && (f.predicate == nil || f.predicate(object, g))
}
func (f ObjectFilter) Text() string { return f.label }

// Cards adapts a characteristic filter for a card query.
func (f ObjectFilter) Cards() CardFilter {
	return NewCardFilter(f.label, func(c *Card) bool { return f.Match(c, nil) })
}

// Permanents adapts a characteristic filter for a battlefield query.
func (f ObjectFilter) Permanents() PermanentFilter {
	return NewPermanentFilter(f.label, func(p *Permanent, g *Game) bool { return f.Match(p, g) })
}

// ObjectHasType tests the current card type.
func ObjectHasType(kind CardType) ObjectFilter {
	return CharacteristicFilter(kind.String(), func(o ObjectView, _ GameReader) bool { return o.HasType(kind) })
}

// ObjectHasSuperType tests the current supertype.
func ObjectHasSuperType(superType SuperType) ObjectFilter {
	return CharacteristicFilter(superType.String(), func(o ObjectView, _ GameReader) bool { return o.HasSuperType(superType) })
}

// ObjectHasSubtype tests the current subtype.
func ObjectHasSubtype(subtype string) ObjectFilter {
	return CharacteristicFilter(subtype, func(o ObjectView, _ GameReader) bool { return o.HasSubType(subtype) })
}

// ObjectHasKeyword tests whether the object has the given keyword.
func ObjectHasKeyword(kw Keyword) ObjectFilter {
	return CharacteristicFilter(kw.String(), func(o ObjectView, _ GameReader) bool { return o.HasKeyword(kw) })
}

// ObjectHasColor tests the current color.
func ObjectHasColor(color Color) ObjectFilter {
	return CharacteristicFilter(color.String(), func(o ObjectView, _ GameReader) bool { return slices.Contains(o.Colors(), color) })
}

// ObjectNot negates a filter.
func ObjectNot(filter ObjectFilter) ObjectFilter {
	return CharacteristicFilter("not "+filter.label, func(o ObjectView, g GameReader) bool {
		return !filter.Match(o, g)
	})
}

// ObjectOr requires at least one supplied filter to match.
func ObjectOr(filters ...ObjectFilter) ObjectFilter {
	specifications := slices.Clone(filters)
	return CharacteristicFilter("any characteristics", func(o ObjectView, g GameReader) bool {
		for _, f := range specifications {
			if f.Match(o, g) {
				return true
			}
		}
		return false
	})
}

// ObjectAnd requires all supplied filters to match.
func ObjectAnd(filters ...ObjectFilter) ObjectFilter {
	specifications := slices.Clone(filters)
	return CharacteristicFilter("all characteristics", func(o ObjectView, g GameReader) bool {
		for _, f := range specifications {
			if !f.Match(o, g) {
				return false
			}
		}
		return true
	})
}
