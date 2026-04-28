package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// Uncounterable is the engine support for spells that "can't be countered."
// Two flavors are supported:
//
//  1. Per-card intrinsic uncounterable, attached via [WithUncounterable].
//     Used for cards whose own Oracle text says "This spell can't be
//     countered" (Allosaurus Shepherd's first ability, Veil of Summer's
//     defended spells, etc.).
//
//  2. Static uncounterable filters registered by a permanent on the
//     battlefield via [RegisterUncounterableStatic]. Used for cards like
//     Allosaurus Shepherd's second ability ("Green spells you control
//     can't be countered") or Vexing Shusher.
//
// Both flavors are consulted before the engine actually removes a spell
// from the stack via [Game.CounterSpellOnStack]. If either matches, the
// counter has no effect (the spell stays on the stack).

// uncounterableMarker is set as a CardOption flag on BaseCard via the
// attrSeeds map under a sentinel key. We use a dedicated bool field
// instead so the flag is preserved across CloneFrom/Copy.

// WithUncounterable marks a card as inherently uncounterable. This works
// regardless of the card's color or type — it applies to the spell once
// it is on the stack.
func WithUncounterable() CardOption {
	return func(c *BaseCard) { c.uncounterable = true }
}

// IsUncounterable reports whether the card itself was registered as
// uncounterable (the per-card flag — does not consult static filters).
func (c *BaseCard) IsUncounterable() bool { return c.uncounterable }

// UncounterableFilter is a static-effect filter: given the game and a
// stack object representing a spell, returns true if that spell can't be
// countered while the source is on the battlefield.
type UncounterableFilter func(g *Game, obj *StackObject) bool

// uncounterableEntry is a registered static filter installed by a
// permanent on the battlefield (e.g. Allosaurus Shepherd's second
// ability).
type uncounterableEntry struct {
	SourceID uuid.UUID
	Filter   UncounterableFilter
}

// RegisterUncounterableStatic returns a continuous effect that, while the
// source permanent is on the battlefield, registers a filter declaring
// that any spell matching it cannot be countered.
//
// Example: Allosaurus Shepherd — "Green spells you control can't be
// countered."
//
//	WithStaticAbility(RegisterUncounterableStatic(func(g *Game, obj *StackObject) bool {
//	    perm := g.FindPermanent(sourceID) // capture sourceID via closure if needed
//	    return obj.Controller == perm.Controller && cardIsGreen(obj.Card)
//	}))
//
// The filter receives the live stack object so it can inspect the
// spell's controller, card type, color, and so on.
func RegisterUncounterableStatic(filter UncounterableFilter) ContinuousEffect {
	return FuncContinuousEffect(LayerAbility, WhileOnBattlefield, func(g *Game, sourceID uuid.UUID) error {
		g.effects.Rules.AddUncounterableFilter(sourceID, filter)
		return nil
	})
}

// IsSpellUncounterable reports whether the stack object — typically the
// target of a counter effect — is currently uncounterable. Consults both
// the per-card intrinsic flag and any registered static filters.
func (g *Game) IsSpellUncounterable(obj *StackObject) bool {
	if obj == nil || obj.IsAbility {
		return false
	}
	if bc, ok := obj.Card.(*BaseCard); ok && bc.uncounterable {
		return true
	}
	if obj.Card != nil {
		if u, ok := obj.Card.(interface{ IsUncounterable() bool }); ok && u.IsUncounterable() {
			return true
		}
	}
	for _, e := range g.effects.Rules.UncounterableFilters {
		if e.Filter == nil {
			continue
		}
		if e.Filter(g, obj) {
			return true
		}
	}
	return false
}
