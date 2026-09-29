package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// LifeGainModifierFn rewrites the proposed life-gain amount for a given
// gaining player. Multiple modifiers compose in registration order.
// Cloned games share the function. Read game state from g and do not change
// captured state.
type LifeGainModifierFn func(g *Game, gainingPlayerID uuid.UUID, amount int) int

// lifeGainModifierEntry pairs a life-gain modifier with its source
// permanent. Entries whose source has left the battlefield are skipped
// during ApplyLifeGainModifiers.
type lifeGainModifierEntry struct {
	sourceID          uuid.UUID
	sourceIncarnation uuid.UUID
	fn                LifeGainModifierFn
}

// AddLifeGainModifier registers a life-gain-amount modifier scoped to the
// given source permanent's current battlefield incarnation. Each modifier is applied
// (in registration order) when the engine resolves a life-gain — see
// ApplyLifeGainModifiers. Auto-cleared whenever the source permanent
// leaves the battlefield.
//
// Example: Rhox Faithmender — "If you would gain life, you gain twice
// that much life instead." The card registers a modifier that, when the
// gaining player is the source's controller, returns 2*amount.
func (g *Game) AddLifeGainModifier(sourceID uuid.UUID, fn LifeGainModifierFn) {
	var inc uuid.UUID
	if source := g.FindPermanentIncludingPhased(sourceID); source != nil {
		inc = source.incarnationID
	}
	g.lifeGainModifiers = append(g.lifeGainModifiers, lifeGainModifierEntry{
		sourceID:          sourceID,
		sourceIncarnation: inc,
		fn:                fn,
	})
}

// LifeGainModifierFactory builds a per-instance LifeGainModifierFn given the
// source permanent's ID and controller.
type LifeGainModifierFactory func(sourceID, controllerID uuid.UUID) LifeGainModifierFn

type staticLifeGainReplacement struct {
	BaseAbility
	factory LifeGainModifierFactory
}

func (a *staticLifeGainReplacement) inspectDefinition() definitionNode {
	n := definitionNode{Role: "static-life-gain-replacement", Text: "modify life gain amount"}
	if a.factory == nil {
		n.problem = "supply a life gain modifier factory"
	}
	return n
}

// WithLifeGainReplacement adds a static life-gain modifier. It applies immediately while
// the source has this ability on the battlefield. Phasing suspends the ability.
// The factory receives the source's current controller for each life-gain action.
// Static modifiers follow registered modifiers, in battlefield and ability order.
// Per CR 614.1a this is a static ability generating a replacement effect, not a triggered ability.
func WithLifeGainReplacement(factory LifeGainModifierFactory) CardOption {
	return func(c *BaseCard) {
		c.AddAbility(&staticLifeGainReplacement{
			BaseAbility: BaseAbility{id: uuid.New(), abilityType: AbilityStatic},
			factory:     factory,
		})
	}
}

// DoubleLifeGainForController is a LifeGainModifierFactory for
// "If you would gain life, you gain twice that much life instead"
// (Rhox Faithmender, Boon Reflection).
// It reads the current controller from the supplied game if available.
func DoubleLifeGainForController(sourceID, controllerID uuid.UUID) LifeGainModifierFn {
	return func(g *Game, gainingID uuid.UUID, amount int) int {
		ctrl := controllerID
		if g != nil {
			if source := g.FindPermanent(sourceID); source != nil {
				ctrl = source.ControllerID()
			}
		}
		if gainingID == ctrl {
			return amount * 2
		}
		return amount
	}
}

// ApplyLifeGainModifiers runs all registered life-gain modifiers and static abilities
// (whose source is still on the battlefield) on the given proposed amount and
// returns the resulting amount. Called by PlayerGainLife before the
// life is actually awarded.
func (g *Game) ApplyLifeGainModifiers(gainingPlayerID uuid.UUID, amount int) int {
	if amount <= 0 {
		return amount
	}
	out := amount
	live := g.lifeGainModifiers[:0]
	for _, m := range g.lifeGainModifiers {
		source := g.FindPermanentIncludingPhased(m.sourceID)
		if source == nil || (m.sourceIncarnation != uuid.Nil && source.incarnationID != m.sourceIncarnation) {
			continue
		}
		live = append(live, m)
		if source.PhasedOut {
			continue
		}
		out = m.fn(g, gainingPlayerID, out)
	}
	g.lifeGainModifiers = live
	for _, source := range g.zones.Battlefield() {
		if source.PhasedOut {
			continue
		}
		for _, ability := range source.RuntimeAbilities {
			if modifier, ok := UnwrapAbility(ability).(*staticLifeGainReplacement); ok {
				out = modifier.factory(source.ID(), source.ControllerID())(g, gainingPlayerID, out)
			}
		}
	}
	return out
}
