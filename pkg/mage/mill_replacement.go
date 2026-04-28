package mage

import "github.com/google/uuid"

// MillModifierFn rewrites the mill amount for a given milled player.
// It receives the milling player and the proposed amount, and returns the
// new amount. Multiple modifiers compose in registration order.
type MillModifierFn func(g *Game, milledPlayerID uuid.UUID, amount int) int

// millModifierEntry pairs a mill-amount modifier with its source permanent.
// Entries whose source has left the battlefield are skipped during apply.
type millModifierEntry struct {
	sourceID uuid.UUID
	fn       MillModifierFn
}

// AddMillReplacement registers a mill-amount modifier scoped to the given
// source permanent. Each modifier is applied (in registration order) when the
// engine resolves a mill — see ApplyMillModifiers. Auto-cleared whenever the
// source permanent leaves the battlefield.
//
// Example: Bruvac the Grandiloquent — "If an opponent would mill one or more
// cards, they mill twice that many cards instead." The card registers a
// modifier that, when the milled player is an opponent of Bruvac's
// controller, returns 2*amount.
func (g *Game) AddMillReplacement(sourceID uuid.UUID, fn MillModifierFn) {
	g.millModifiers = append(g.millModifiers, millModifierEntry{sourceID: sourceID, fn: fn})
}

// MillModifierFactory builds a per-instance MillModifierFn given the source
// permanent's ID and controller (captured at ETB time). Used by
// WithMillReplacement to express "an opponent would mill" / "you would mill"
// scoping without per-card FuncEffect boilerplate.
type MillModifierFactory func(sourceID, controllerID uuid.UUID) MillModifierFn

// WithMillReplacement registers a mill replacement on this permanent at ETB,
// auto-cleared when it leaves the battlefield. Per CR 701.13b mill is a
// game action whose card count is replaceable; per CR 614.6 these stack
// additively in registration order.
func WithMillReplacement(factory MillModifierFactory) CardOption {
	return func(c *BaseCard) {
		c.AddAbility(EntersBattlefieldTrigger(
			FuncEffect("install mill modifier",
				EffectProperties{},
				func(g *Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
					g.AddMillReplacement(sourceID, factory(sourceID, controller))
					return nil
				},
			),
			false,
		))
	}
}

// DoubleMillForOpponents is a MillModifierFactory for
// "If an opponent would mill one or more cards, they mill twice that many
// cards instead" (Bruvac the Grandiloquent).
func DoubleMillForOpponents(_, controllerID uuid.UUID) MillModifierFn {
	return func(_ *Game, milledID uuid.UUID, amount int) int {
		if milledID == controllerID || amount < 1 {
			return amount
		}
		return amount * 2
	}
}

// ApplyMillModifiers runs all registered mill modifiers (whose source is
// still on the battlefield) on the given proposed amount and returns the
// resulting amount. Called by the mill execution path before any cards are
// moved.
func (g *Game) ApplyMillModifiers(milledPlayerID uuid.UUID, amount int) int {
	if amount <= 0 {
		return amount
	}
	out := amount
	live := g.millModifiers[:0]
	for _, m := range g.millModifiers {
		if g.FindPermanent(m.sourceID) == nil {
			continue
		}
		live = append(live, m)
		out = m.fn(g, milledPlayerID, out)
	}
	g.millModifiers = live
	return out
}
