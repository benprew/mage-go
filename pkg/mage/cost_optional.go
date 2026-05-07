package mage

import (
	"github.com/google/uuid"
)

// optionalCost wraps an inner cost in a "may pay" gate. CanPay is always
// true (you can always decline). At Pay time, the controller is prompted
// via ChooseMayAbility; if they accept and the inner cost is payable, the
// inner cost is paid and the source's "optional cost paid" flag is set on
// the Game so resolution-time effects can branch on it.
//
// Used by spells like Draconic Roar ("As an additional cost, you may
// reveal a Dragon card from your hand") and similar may-pay additional
// costs whose payment unlocks a stronger resolution branch.
type optionalCost struct {
	inner  Cost
	prompt string
}

// OptionalCost creates a "may-pay" wrapper over an inner cost. The prompt
// is shown to the controller via ChooseMayAbility. After payment, query
// the outcome at resolution via Game.LastCostOptionalPaid(sourceID).
func OptionalCost(inner Cost, prompt string) Cost {
	return &optionalCost{inner: inner, prompt: prompt}
}

func (c *optionalCost) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	return true
}

func (c *optionalCost) Pay(sourceID, controller uuid.UUID, g *Game) error {
	g.setOptionalCostPaid(sourceID, false)
	if !c.inner.CanPay(sourceID, controller, g) {
		return nil
	}
	p := g.GetPlayer(controller)
	if p == nil {
		return nil
	}
	if !p.ChooseMayAbility(c.prompt) {
		return nil
	}
	if err := c.inner.Pay(sourceID, controller, g); err != nil {
		return nil
	}
	g.setOptionalCostPaid(sourceID, true)
	return nil
}

func (c *optionalCost) Text() string {
	return "you may " + c.inner.Text()
}

func (g *Game) setOptionalCostPaid(sourceID uuid.UUID, paid bool) {
	if g.optionalCostPaid == nil {
		g.optionalCostPaid = make(map[uuid.UUID]bool)
	}
	g.optionalCostPaid[sourceID] = paid
}

// LastCostOptionalPaid reports whether the most recent OptionalCost
// attached to the given source paid its inner cost. Effects resolving from
// that source can branch on this to apply the "if you do, ___" half of an
// optional additional cost (e.g. Draconic Roar's bonus damage).
func (g *Game) LastCostOptionalPaid(sourceID uuid.UUID) bool {
	if g.optionalCostPaid == nil {
		return false
	}
	return g.optionalCostPaid[sourceID]
}

// ClearOptionalCostPaid clears the "paid" flag for a source. Called by
// the engine when a spell or ability finishes resolving so the flag does
// not leak into a future cast.
func (g *Game) ClearOptionalCostPaid(sourceID uuid.UUID) {
	if g.optionalCostPaid == nil {
		return
	}
	delete(g.optionalCostPaid, sourceID)
}
