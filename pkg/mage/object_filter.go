package mage

import (
	"reflect"
	"slices"
	"strings"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ObjectView supplies characteristics that are common to cards and permanents.
type ObjectView interface {
	ViewID() uuid.UUID
	ViewName() string
	ViewController() uuid.UUID
	ViewOwner() uuid.UUID
	ViewHasType(CardType) bool
	ViewHasSuperType(SuperType) bool
	ViewHasSubType(string) bool
	ViewHasKeyword(Keyword) bool
	ViewColors() []Color
	ViewManaCost() ManaCost
	ViewManaValue() int
	ViewPower() int
	ViewToughness() int
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
	kind         string
	children     []ObjectFilter
	comparison   *ValueComparison
	cardType     CardType
	label        string
	creatureOnly bool
	predicate    func(ObjectView, AbilityContext, GameReader) bool
}

// CharacteristicFilter creates a filter that can be used in any zone.
func CharacteristicFilter(label string, predicate func(ObjectView, GameReader) bool) ObjectFilter {
	return ContextObjectFilter(label, func(o ObjectView, _ AbilityContext, g GameReader) bool { return predicate(o, g) })
}

// ContextObjectFilter creates a filter with source and controller bindings.
func ContextObjectFilter(label string, predicate func(ObjectView, AbilityContext, GameReader) bool) ObjectFilter {
	return ObjectFilter{label: label, predicate: predicate}
}

// Match rejects nil objects before it evaluates the predicate.
func (f ObjectFilter) Match(object ObjectView, g GameReader, contexts ...AbilityContext) bool {
	if object == nil {
		return false
	}
	value := reflect.ValueOf(object)
	if value.Kind() == reflect.Pointer && value.IsNil() {
		return false
	}
	var ctx AbilityContext
	if len(contexts) > 0 {
		ctx = contexts[0]
	}
	if p, ok := object.(*Permanent); ok {
		object = livePermanentView{p: p, g: g}
	}
	matched, known := f.evaluate(object, ctx, g, TargetingContext{})
	return !known || matched
}
func (f ObjectFilter) Text() string { return f.label }

// IsZero reports whether the filter is the zero value and matches all objects.
func (f ObjectFilter) IsZero() bool { return f.kind == "" && f.predicate == nil }

// Permanents adapts a characteristic filter for a battlefield query.
func (f ObjectFilter) Permanents(contexts ...AbilityContext) PermanentFilter {
	var ctx AbilityContext
	if len(contexts) > 0 {
		ctx = contexts[0]
	}
	return NewPermanentFilter(f.label, func(p *Permanent, g *Game) bool { return f.Match(p, g, ctx) })
}

// ObjectHasType tests the current card type.
func ObjectHasType(kind CardType) ObjectFilter {
	filter := CharacteristicFilter(kind.String(), func(o ObjectView, _ GameReader) bool { return o.ViewHasType(kind) })
	filter.kind, filter.cardType = "IsCardType", kind
	filter.creatureOnly = kind == TypeCreature
	return filter
}

// ObjectHasSuperType tests the current supertype.
func ObjectHasSuperType(superType SuperType) ObjectFilter {
	return CharacteristicFilter(superType.String(), func(o ObjectView, _ GameReader) bool { return o.ViewHasSuperType(superType) })
}

// ObjectHasSubtype tests the current subtype.
func ObjectHasSubtype(subtype string) ObjectFilter {
	return CharacteristicFilter(subtype, func(o ObjectView, _ GameReader) bool { return o.ViewHasSubType(subtype) })
}

// ObjectHasKeyword tests whether the object has the given keyword.
func ObjectHasKeyword(kw Keyword) ObjectFilter {
	return CharacteristicFilter(kw.String(), func(o ObjectView, _ GameReader) bool { return o.ViewHasKeyword(kw) })
}

// ObjectHasColor tests the current color.
func ObjectHasColor(color Color) ObjectFilter {
	return CharacteristicFilter(color.String(), func(o ObjectView, _ GameReader) bool { return slices.Contains(o.ViewColors(), color) })
}

// ObjectNamed tests whether the object has the given name.
func ObjectNamed(name string) ObjectFilter {
	return CharacteristicFilter(name, func(o ObjectView, _ GameReader) bool { return o.ViewName() == name })
}

// ObjectNot negates a filter.
func ObjectNot(filter ObjectFilter) ObjectFilter {
	label := "not " + filter.label
	if filter.label == "" {
		label = ""
	}
	return ObjectFilter{kind: "Not", label: label, children: []ObjectFilter{filter}}
}

// ObjectOr requires at least one supplied filter to match.
func ObjectOr(filters ...ObjectFilter) ObjectFilter {
	parts := make([]string, 0, len(filters))
	for _, filter := range filters {
		if filter.label != "" {
			parts = append(parts, filter.label)
		}
	}
	label := "any characteristics"
	if len(parts) > 0 {
		label = strings.Join(parts, " or ")
	}
	result := ObjectFilter{kind: "Or", label: label, children: slices.Clone(filters), creatureOnly: len(filters) > 0}
	for _, filter := range filters {
		result.creatureOnly = result.creatureOnly && filter.creatureOnly
	}
	return result
}

// ObjectAnd requires all supplied filters to match.
func ObjectAnd(filters ...ObjectFilter) ObjectFilter {
	parts := make([]string, 0, len(filters))
	for _, filter := range filters {
		if filter.label != "" {
			parts = append(parts, filter.label)
		}
	}
	label := "all characteristics"
	if len(parts) > 0 {
		label = strings.Join(parts, " ")
	}
	result := ObjectFilter{kind: "And", label: label, children: slices.Clone(filters)}
	for _, filter := range filters {
		result.creatureOnly = result.creatureOnly || filter.creatureOnly
	}
	return result
}

// Common object characteristic filters.
var (
	IsCreatureCard         = ObjectHasType(TypeCreature)
	IsArtifactCard         = ObjectHasType(TypeArtifact)
	IsEnchantmentCard      = ObjectHasType(TypeEnchantment)
	IsInstantCard          = ObjectHasType(TypeInstant)
	IsSorceryCard          = ObjectHasType(TypeSorcery)
	IsInstantOrSorceryCard = ObjectOr(ObjectHasType(TypeInstant), ObjectHasType(TypeSorcery))
	IsPermanentCard        = ObjectOr(
		ObjectHasType(TypeArtifact),
		ObjectHasType(TypeCreature),
		ObjectHasType(TypeEnchantment),
		ObjectHasType(TypeLand),
		ObjectHasType(TypePlaneswalker),
	)
)

// IsCardType tests the current card type.
func IsCardType(t CardType) ObjectFilter { return ObjectHasType(t) }

// OtherThan excludes objects selected in the ability context.
func OtherThan(sel ObjectSelector) ObjectFilter {
	return ContextObjectFilter("other", func(o ObjectView, ctx AbilityContext, g GameReader) bool {
		return !slices.Contains(sel.Select(filterEffectContext(ctx, g)), o.ViewID())
	})
}

// ObjectControlledBy tests the controller against a player selector.
func ObjectControlledBy(sel PlayerSelector) ObjectFilter {
	return ContextObjectFilter("controlled by "+sel.Text(), func(o ObjectView, ctx AbilityContext, g GameReader) bool {
		controller := o.ViewController()
		return controller != uuid.Nil() && slices.Contains(sel.Select(filterEffectContext(ctx, g)), controller)
	})
}

// ObjectOwnedBy tests the owner against a player selector.
func ObjectOwnedBy(sel PlayerSelector) ObjectFilter {
	return ContextObjectFilter("owned by "+sel.Text(), func(o ObjectView, ctx AbilityContext, g GameReader) bool {
		owner := o.ViewOwner()
		return owner != uuid.Nil() && slices.Contains(sel.Select(filterEffectContext(ctx, g)), owner)
	})
}

func filterEffectContext(ctx AbilityContext, reader GameReader) *EffectContext {
	g, _ := reader.(*Game)
	return NewEffectContext(g, ctx.SourceID, ctx.Controller, nil)
}

func (c *Card) ViewID() uuid.UUID                 { return c.ID() }
func (c *Card) ViewName() string                  { return c.Name() }
func (c *Card) ViewController() uuid.UUID         { return uuid.Nil() }
func (c *Card) ViewOwner() uuid.UUID              { return c.Owner() }
func (c *Card) ViewHasType(t CardType) bool       { return c.HasType(t) }
func (c *Card) ViewHasSuperType(t SuperType) bool { return c.HasSuperType(t) }
func (c *Card) ViewHasSubType(t string) bool      { return c.HasSubType(t) }
func (c *Card) ViewHasKeyword(t Keyword) bool     { return c.HasKeyword(t) }
func (c *Card) ViewColors() []Color               { return c.Colors() }
func (c *Card) ViewManaCost() ManaCost            { return c.ManaCost() }
func (c *Card) ViewManaValue() int                { return c.ManaValue() }
func (c *Card) ViewPower() int                    { return c.Power() }
func (c *Card) ViewToughness() int                { return c.Toughness() }

func (p *Permanent) ViewID() uuid.UUID                 { return p.ID() }
func (p *Permanent) ViewName() string                  { return p.Name() }
func (p *Permanent) ViewController() uuid.UUID         { return p.ControllerID() }
func (p *Permanent) ViewOwner() uuid.UUID              { return (livePermanentView{p: p}).ViewOwner() }
func (p *Permanent) ViewHasType(t CardType) bool       { return p.HasType(t) }
func (p *Permanent) ViewHasSuperType(t SuperType) bool { return p.HasSuperType(t) }
func (p *Permanent) ViewHasSubType(t string) bool      { return p.HasSubType(t) }
func (p *Permanent) ViewHasKeyword(t Keyword) bool     { return p.HasKeyword(t) }
func (p *Permanent) ViewColors() []Color               { return p.Colors() }
func (p *Permanent) ViewManaCost() ManaCost {
	if p.FaceDown {
		return ManaCost{}
	}
	return p.Card.ManaCost()
}
func (p *Permanent) ViewManaValue() int { return p.ManaValue() }
func (p *Permanent) ViewPower() int     { return p.CurrentPower(nil) + p.powerBonus }
func (p *Permanent) ViewToughness() int { return p.CurrentToughness(nil) + p.toughBonus }

type spellObjectView struct {
	*Card
	controller uuid.UUID
}

func (s spellObjectView) ViewController() uuid.UUID { return s.controller }

type stackObjectView struct {
	*Card
	object *StackObject
}

func (s stackObjectView) ViewID() uuid.UUID         { return s.object.SourceID }
func (s stackObjectView) ViewController() uuid.UUID { return s.object.Controller }
func (s stackObjectView) ViewManaValue() int        { return s.object.ManaValue() }
func (s stackObjectView) ViewColors() []Color {
	if s.object.ColorOverride != nil {
		return slices.Clone(*s.object.ColorOverride)
	}
	return s.Colors()
}
