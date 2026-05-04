package mage

import "github.com/google/uuid"

// LifeGainModifierFn rewrites the proposed life-gain amount for a given
// gaining player. Multiple modifiers compose in registration order.
type LifeGainModifierFn func(g *Game, gainingPlayerID uuid.UUID, amount int) int

// lifeGainModifierEntry pairs a life-gain modifier with its source
// permanent. Entries whose source has left the battlefield are skipped
// during ApplyLifeGainModifiers.
type lifeGainModifierEntry struct {
	sourceID uuid.UUID
	fn       LifeGainModifierFn
}

// AddLifeGainModifier registers a life-gain-amount modifier scoped to the
// given source permanent. Each modifier is applied (in registration
// order) when the engine resolves a life-gain — see
// ApplyLifeGainModifiers. Auto-cleared whenever the source permanent
// leaves the battlefield.
//
// Example: Rhox Faithmender — "If you would gain life, you gain twice
// that much life instead." The card registers a modifier that, when the
// gaining player is the source's controller, returns 2*amount.
func (g *Game) AddLifeGainModifier(sourceID uuid.UUID, fn LifeGainModifierFn) {
	g.lifeGainModifiers = append(g.lifeGainModifiers, lifeGainModifierEntry{sourceID: sourceID, fn: fn})
}

// LifeGainModifierFactory builds a per-instance LifeGainModifierFn given the
// source permanent's ID and controller (captured at ETB time). Used by
// WithLifeGainReplacement so cards can express "your" / "an opponent's"
// scoping without re-implementing the ETB+register boilerplate.
type LifeGainModifierFactory func(sourceID, controllerID uuid.UUID) LifeGainModifierFn

// WithLifeGainReplacement registers a life-gain replacement on this permanent
// at ETB. The factory captures the source/controller so card text like "If
// you would gain life..." (Rhox Faithmender) and "If an opponent would gain
// life..." can be expressed without per-card FuncEffect boilerplate. The
// modifier auto-clears when the permanent leaves the battlefield (see
// ApplyLifeGainModifiers). Per CR 614.6 multiple replacements stack
// additively in registration order; CR 616 governs ordering when more than
// one replacement applies to the same event.
func WithLifeGainReplacement(factory LifeGainModifierFactory) CardOption {
	return func(c *BaseCard) {
		c.AddAbility(EntersBattlefieldTrigger(
			FuncEffect("install life-gain modifier",
				EffectProperties{Outcome: OutcomeBenefit},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					g.AddLifeGainModifier(sourceID, factory(sourceID, controller))
					return nil
				},
			),
			false,
		))
	}
}

// DoubleLifeGainForController is a LifeGainModifierFactory for
// "If you would gain life, you gain twice that much life instead"
// (Rhox Faithmender, Boon Reflection).
func DoubleLifeGainForController(_, controllerID uuid.UUID) LifeGainModifierFn {
	return func(_ *Game, gainingID uuid.UUID, amount int) int {
		if gainingID == controllerID {
			return amount * 2
		}
		return amount
	}
}

// ApplyLifeGainModifiers runs all registered life-gain modifiers (whose
// source is still on the battlefield) on the given proposed amount and
// returns the resulting amount. Called by PlayerGainLife before the
// life is actually awarded.
func (g *Game) ApplyLifeGainModifiers(gainingPlayerID uuid.UUID, amount int) int {
	if amount <= 0 {
		return amount
	}
	out := amount
	live := g.lifeGainModifiers[:0]
	for _, m := range g.lifeGainModifiers {
		if g.FindPermanent(m.sourceID) == nil {
			continue
		}
		live = append(live, m)
		out = m.fn(g, gainingPlayerID, out)
	}
	g.lifeGainModifiers = live
	return out
}
