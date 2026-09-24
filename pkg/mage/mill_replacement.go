package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// MillModifierFn rewrites the mill amount for a given milled player.
// It receives the milling player and the proposed amount, and returns the
// new amount. Multiple modifiers compose in registration order.
// Game clones share this function. Read state from the supplied game and do
// not change captured state.
type MillModifierFn func(g *Game, milledPlayerID uuid.UUID, amount int) int

// millModifierEntry pairs a mill-amount modifier with its source permanent.
// Entries whose source has left the battlefield are skipped during apply.
type millModifierEntry struct {
	sourceID          uuid.UUID
	sourceIncarnation uuid.UUID
	fn                MillModifierFn
}

// AddMillReplacement registers a mill-amount modifier scoped to the given
// source permanent's current battlefield incarnation. An absent source does
// not register a modifier. Modifiers apply in registration order. Phasing
// suspends a modifier; leaving the battlefield ends it.
//
// Example: Bruvac the Grandiloquent — "If an opponent would mill one or more
// cards, they mill twice that many cards instead." The card registers a
// modifier that, when the milled player is an opponent of Bruvac's
// controller, returns 2*amount.
func (g *Game) AddMillReplacement(sourceID uuid.UUID, fn MillModifierFn) {
	source := g.FindPermanentIncludingPhased(sourceID)
	if source == nil {
		return
	}
	g.millModifiers = append(g.millModifiers, millModifierEntry{
		sourceID: sourceID, sourceIncarnation: source.incarnationID, fn: fn,
	})
}

// MillModifierFactory builds a modifier with the current source and controller.
// WithMillReplacement calls the factory for each proposed positive mill amount.
// Factories and their modifiers must not change captured state.
type MillModifierFactory func(sourceID, controllerID uuid.UUID) MillModifierFn

type staticMillReplacement struct {
	BaseAbility
	factory MillModifierFactory
}

func (a *staticMillReplacement) inspectDefinition() definitionNode {
	n := definitionNode{Role: "static-mill-replacement", Text: "modify mill amount"}
	if a.factory == nil {
		n.problem = "supply a mill modifier factory"
	}
	return n
}

// WithMillReplacement adds a static mill modifier. It applies immediately while
// the source has this ability on the battlefield. Phasing suspends the ability.
// The factory receives the source's current controller for each mill action.
// Static modifiers follow registered modifiers, in battlefield and ability order.
// This legacy helper does not implement the replacement choices of CR 616.
func WithMillReplacement(factory MillModifierFactory) CardOption {
	return func(c *BaseCard) {
		c.AddAbility(&staticMillReplacement{
			BaseAbility: BaseAbility{id: uuid.New(), abilityType: AbilityStatic},
			factory:     factory,
		})
	}
}

// DoubleMillForOpponents is a MillModifierFactory for
// "If an opponent would mill one or more cards, they mill twice that many
// cards instead" (Bruvac the Grandiloquent).
// It reads the current controller from the supplied game. The controller
// argument is retained for compatibility with MillModifierFactory.
func DoubleMillForOpponents(sourceID, _ uuid.UUID) MillModifierFn {
	return func(g *Game, milledID uuid.UUID, amount int) int {
		source := g.FindPermanent(sourceID)
		if source == nil || milledID == source.ControllerID() || amount < 1 {
			return amount
		}
		return amount * 2
	}
}

// ApplyMillModifiers applies live registered modifiers and current static
// abilities to a proposed mill amount before the mill executor moves cards.
func (g *Game) ApplyMillModifiers(milledPlayerID uuid.UUID, amount int) int {
	if amount <= 0 {
		return amount
	}
	out := amount
	live := g.millModifiers[:0]
	for _, m := range g.millModifiers {
		source := g.FindPermanentIncludingPhased(m.sourceID)
		if source == nil || source.incarnationID != m.sourceIncarnation {
			continue
		}
		live = append(live, m)
		if source.PhasedOut {
			continue
		}
		out = m.fn(g, milledPlayerID, out)
	}
	g.millModifiers = live
	for _, source := range g.zones.Battlefield() {
		if source.PhasedOut {
			continue
		}
		for _, ability := range source.RuntimeAbilities {
			if modifier, ok := UnwrapAbility(ability).(*staticMillReplacement); ok {
				out = modifier.factory(source.ID(), source.ControllerID())(g, milledPlayerID, out)
			}
		}
	}
	return out
}
