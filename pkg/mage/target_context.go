package mage

import (
	"slices"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// TargetingContext distinguishes an announced zero from an unspecified X.
type TargetingContext struct {
	X          int
	XSpecified bool
}

// ValueComparison retains the operator and value for inspection.
type ValueComparison struct {
	Operator string
	Value    ValueSource
}

// EqualTo compares a characteristic with a value.
func EqualTo(value ValueSource) ValueComparison {
	return ValueComparison{Operator: "EqualTo", Value: value}
}

// ManaValueIs compares an object's mana value with a value source.
func ManaValueIs(comparison ValueComparison) ObjectFilter {
	if comparison.Operator != "EqualTo" || comparison.Value == nil {
		panic("mage: invalid mana value comparison")
	}
	return ObjectFilter{kind: "ManaValueIs", label: "mana value", comparison: &comparison}
}

// Kind identifies a structured filter operation. An empty kind is a predicate.
func (f ObjectFilter) Kind() string { return f.kind }

// Children returns a copy of the filter operands.
func (f ObjectFilter) Children() []ObjectFilter { return slices.Clone(f.children) }

// ManaValueComparison returns the comparison for a mana value filter.
func (f ObjectFilter) ManaValueComparison() (ValueComparison, bool) {
	if f.comparison == nil {
		return ValueComparison{}, false
	}
	return *f.comparison, true
}

// CardType returns the type tested by an IsCardType filter.
func (f ObjectFilter) CardType() (CardType, bool) { return f.cardType, f.kind == "IsCardType" }

// UsesX reports a structured comparison that reads the announced X.
func (f ObjectFilter) UsesX() bool {
	if f.comparison != nil && valueUsesX(f.comparison.Value) {
		return true
	}
	return slices.ContainsFunc(f.children, ObjectFilter.UsesX)
}

func valueUsesX(value ValueSource) bool {
	switch v := value.(type) {
	case xValue, halfXRoundedDown, halfXRoundedUp:
		return true
	case addValue:
		return valueUsesX(v.a) || valueUsesX(v.b)
	case mulValue:
		return valueUsesX(v.a) || valueUsesX(v.b)
	case halfRoundUpValue:
		return valueUsesX(v.inner)
	default:
		return false
	}
}

func (f ObjectFilter) evaluate(o ObjectView, ability AbilityContext, g GameReader, ctx TargetingContext) (matches, determined bool) {
	switch f.kind {
	case "ManaValueIs":
		if f.UsesX() && !ctx.XSpecified {
			return false, false
		}
		effect := filterEffectContext(ability, g)
		effect.XValue = ctx.X
		return o.ViewManaValue() == f.comparison.Value.Resolve(effect), true
	case "Not":
		matched, known := f.children[0].evaluate(o, ability, g, ctx)
		return !matched, known
	case "And", "Or":
		known := true
		for _, child := range f.children {
			matched, childKnown := child.evaluate(o, ability, g, ctx)
			if childKnown && matched == (f.kind == "Or") {
				return matched, true
			}
			known = known && childKnown
		}
		return f.kind == "And", known
	default:
		return f.predicate == nil || f.predicate(o, ability, g), true
	}
}

func (f ObjectFilter) matchTarget(p *Permanent, g *Game, ability AbilityContext, ctx TargetingContext) bool {
	matched, known := f.evaluate(livePermanentView{p: p, g: g}, ability, g, ctx)
	return !known || matched
}

// PossibleTargets evaluates a target without changing game state.
func PossibleTargets(target Target, controller uuid.UUID, source *Card, g *Game, ctx TargetingContext) []uuid.UUID {
	switch t := target.(type) {
	case *countBoundTarget:
		return PossibleTargets(t.inner, controller, source, g, ctx)
	case *randomTarget:
		return PossibleTargets(t.inner, controller, source, g, ctx)
	case *opponentChosenTarget:
		return PossibleTargets(t.inner, controller, source, g, ctx)
	case interface {
		PossibleWithContext(uuid.UUID, *Card, *Game, TargetingContext) []uuid.UUID
	}:
		return t.PossibleWithContext(controller, source, g, ctx)
	default:
		return target.Possible(controller, source, g)
	}
}

// ObjectFilter returns the structured characteristic restriction.
func (t *PermanentTarget) ObjectFilter() ObjectFilter { return t.objectFilter }

// TargetsUseX reports whether target characteristics depend on X.
func TargetsUseX(targets []Target) bool {
	for _, target := range targets {
		switch t := target.(type) {
		case *countBoundTarget:
			if TargetsUseX([]Target{t.inner}) {
				return true
			}
		case *randomTarget:
			if TargetsUseX([]Target{t.inner}) {
				return true
			}
		case *opponentChosenTarget:
			if TargetsUseX([]Target{t.inner}) {
				return true
			}
		case interface{ ObjectFilter() ObjectFilter }:
			if t.ObjectFilter().UsesX() {
				return true
			}
		}
	}
	return false
}

func (f ObjectFilter) candidateXValues(g *Game) ([]int, bool) {
	if f.kind == "ManaValueIs" {
		if _, directX := f.comparison.Value.(xValue); !directX {
			return nil, false
		}
		battlefield := g.AllBattlefield()
		values := make([]int, 0, len(battlefield))
		for _, p := range battlefield {
			values = append(values, p.ManaValue())
		}
		return values, true
	}
	if f.kind == "And" {
		for _, child := range f.children {
			if values, finite := child.candidateXValues(g); finite {
				return values, true
			}
		}
	}
	if f.kind == "Or" {
		var values []int
		for _, child := range f.children {
			candidates, finite := child.candidateXValues(g)
			if !finite {
				return nil, false
			}
			values = append(values, candidates...)
		}
		return values, true
	}
	return nil, false
}

func targetCandidateXValues(target Target, g *Game) ([]int, bool) {
	switch t := target.(type) {
	case *countBoundTarget:
		return targetCandidateXValues(t.inner, g)
	case *randomTarget:
		return targetCandidateXValues(t.inner, g)
	case *opponentChosenTarget:
		return targetCandidateXValues(t.inner, g)
	case interface{ ObjectFilter() ObjectFilter }:
		return t.ObjectFilter().candidateXValues(g)
	default:
		return nil, false
	}
}

// SpellTargetOption pairs an affordable X with candidates for every target group.
type SpellTargetOption struct {
	XValue     int
	Candidates [][]uuid.UUID
}

// SpellTargetOptions finds affordable announcements with enough legal targets.
// Supplied values limit X choices for spells without an X target restriction.
func (g *Game) SpellTargetOptions(controller uuid.UUID, card *Card, values ...int) []SpellTargetOption {
	specs := card.CastTargets()
	if !card.ManaCost().HasX {
		values = []int{0}
	} else if len(values) == 0 || TargetsUseX(specs) {
		values = nil
		finite := false
		for _, spec := range specs {
			if spec.Min() == 0 {
				continue
			}
			if candidates, ok := targetCandidateXValues(spec, g); ok {
				values, finite = candidates, true
				break
			}
		}
		if !finite {
			for x, maximum := 0, g.MaxXValue(controller, card.ManaCost(), SpellContextForCard(card)); x <= maximum; x++ {
				values = append(values, x)
			}
		}
		slices.Sort(values)
		values = slices.Compact(values)
	}
	var result []SpellTargetOption
	for _, x := range values {
		if x < 0 || !g.CanAfford(controller, resolvedSpellManaCost(card.ManaCost(), x), SpellContextForCard(card)) {
			continue
		}
		option := SpellTargetOption{XValue: x, Candidates: make([][]uuid.UUID, len(specs))}
		legal := true
		for i, spec := range specs {
			option.Candidates[i] = PossibleTargets(spec, controller, card, g, TargetingContext{X: x, XSpecified: true})
			minimum, _ := TargetBounds(spec, x)
			if len(option.Candidates[i]) < minimum {
				legal = false
				break
			}
		}
		if legal {
			result = append(result, option)
		}
	}
	return result
}
