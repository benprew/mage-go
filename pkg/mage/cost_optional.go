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
	inner       Cost
	prompt      string
	result      *NumberBinding
	capture     bool
	captureCard bool
}

// OptionalCost creates a "may-pay" wrapper over an inner cost. The prompt
// is shown to the controller via ChooseMayAbility. After payment, query
// the outcome at resolution via Game.LastCostOptionalPaid(sourceID).
func OptionalCost(inner Cost, prompt string) Cost {
	return &optionalCost{inner: inner, prompt: prompt}
}

// OptionalCostInto saves 1 if the player pays the cost and 0 otherwise.
func OptionalCostInto(inner Cost, prompt string, result *NumberBinding) Cost {
	return &optionalCost{inner: inner, prompt: prompt, result: result, capture: true}
}

func (c *optionalCost) resolveActionCost(ctx actionCostContext) Cost {
	resolved := *c
	if counter, ok := c.inner.(*removeXCountersCost); ok {
		resolved.inner = counter.resolveActionCost(ctx)
	}
	return &resolved
}

func uncapturedFixedCounterAlternatives(cost Cost) bool {
	switch c := cost.(type) {
	case *removeCountersCost:
		return !c.capture
	case *eitherCost:
		if len(c.options) == 0 {
			return false
		}
		for _, option := range c.options {
			if !uncapturedFixedCounterAlternatives(option) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// OptionalRevealFromHandCost saves the payment choice and the revealed card.
func OptionalRevealFromHandCost(filter CardFilter, label, prompt string, paid *NumberBinding, card *CardBinding) Cost {
	return &optionalCost{
		inner:  RevealFromHandCostWithCard(filter, label, card),
		prompt: prompt, result: paid, capture: true, captureCard: true,
	}
}

func (c *optionalCost) inspectDefinition() definitionNode {
	n := definitionNode{Role: "optional-cost", Text: c.Text(), children: []definitionEdge{{"cost", c.inner}}}
	if c.inner == nil {
		n.problem = "supply a non-nil optional cost"
	}
	if c.capture && c.result == nil {
		n.problem = "supply a non-nil optional payment binding"
	}
	if c.capture && c.prompt == "" {
		n.problem = "supply optional payment prompt text"
	}
	if discard, ok := c.inner.(*discardCost); ok && (discard.captureCards || discard.captureCard) {
		n.problem = "paid discard cards require a direct cost"
	}
	if c.capture && c.inner != nil {
		if _, nested := c.inner.(*optionalCost); nested {
			n.problem = "optional payment result requires a direct cost"
		}
		if _, alternate := c.inner.(*eitherCost); alternate {
			n.problem = "optional payment result requires a direct cost"
		}
		if inspector, ok := c.inner.(definitionInspector); ok && len(inspector.inspectDefinition().bindingWrites) != 0 && !c.captureCard {
			n.problem = "optional payment result cannot wrap another captured cost"
		}
		if c.captureCard {
			if reveal, ok := c.inner.(*revealFromHandCost); !ok || !reveal.capture {
				n.problem = "optional card capture requires a revealed hand-card cost"
			}
		}
	}
	if c.result != nil {
		n.bindingWrites = []savedBinding{c.result}
		n.children = append(n.children, definitionEdge{"result", c.result})
	}
	return n
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
		return err
	}
	g.setOptionalCostPaid(sourceID, true)
	return nil
}

func (c *optionalCost) Text() string {
	if c.inner == nil {
		return "optional cost"
	}
	return "you may " + c.inner.Text()
}

type optionalOutcomeCost struct {
	result *NumberBinding
	paid   bool
}

func (*optionalOutcomeCost) CanPay(uuid.UUID, uuid.UUID, *Game) bool { return true }
func (*optionalOutcomeCost) Pay(uuid.UUID, uuid.UUID, *Game) error   { return nil }
func (*optionalOutcomeCost) Text() string                            { return "optional payment result" }

func (g *Game) setOptionalCostPaid(sourceID uuid.UUID, paid bool) {
	g.trackers.Turn.SetOptionalCostPaid(sourceID, paid)
}

// LastCostOptionalPaid reports whether the most recent OptionalCost
// attached to the given source paid its inner cost. Effects resolving from
// that source can branch on this to apply the "if you do, ___" half of an
// optional additional cost (e.g. Draconic Roar's bonus damage).
func (g *Game) LastCostOptionalPaid(sourceID uuid.UUID) bool {
	return g.trackers.Turn.OptionalCostPaid(sourceID)
}

// ClearOptionalCostPaid clears the "paid" flag for a source. Called by
// the engine when a spell or ability finishes resolving so the flag does
// not leak into a future cast.
func (g *Game) ClearOptionalCostPaid(sourceID uuid.UUID) {
	g.trackers.Turn.ClearOptionalCostPaid(sourceID)
}
