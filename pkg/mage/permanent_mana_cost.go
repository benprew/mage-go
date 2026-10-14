package mage

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type permanentManaCostPayment struct{ subject PermanentRef }

// PermanentManaCostPayment pays the current mana cost of a present permanent.
// The shared transaction locks the cost before mana abilities. X is zero.
// An absent mana cost cannot be paid, even when cost reductions apply.
func PermanentManaCostPayment(subject PermanentRef) Cost {
	return &permanentManaCostPayment{subject: subject}
}

func (c *permanentManaCostPayment) Text() string { return "Pay the permanent's mana cost" }
func (c *permanentManaCostPayment) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cost", Text: c.Text(), children: []definitionEdge{{"subject", c.subject}}}
	if nilDefinition(c.subject) {
		n.problem = "supply a permanent reference"
	}
	return n
}
func (c *permanentManaCostPayment) manaCost(sourceID, controller uuid.UUID, g *Game) (core.ManaCost, bool) {
	if nilDefinition(c.subject) || g.GetPlayer(controller) == nil {
		return core.ManaCost{}, false
	}
	source := g.FindPermanent(sourceID)
	if source == nil || source.PhasedOut {
		return core.ManaCost{}, false
	}
	ctx := &EffectContext{Game: g, SourceID: sourceID, Controller: controller, sourceCaptured: true, sourceIncarnation: source.incarnationID}
	subject, ok := c.subject.resolvePermanent(ctx)
	if !ok || subject.PhasedOut {
		return core.ManaCost{}, false
	}
	card := subject.Card
	if copied, ok := card.(permanentSpellCopyCard); ok {
		card = copied.Card
	}
	presence, ok := card.(interface{ HasManaCost() bool })
	if !ok || !presence.HasManaCost() || subject.FaceDown {
		return core.ManaCost{}, false
	}
	mc := card.ManaCost()
	mc.HasX = false
	mc.XCount = 0
	return mc, true
}
func (c *permanentManaCostPayment) CanPay(sourceID, controller uuid.UUID, g *Game) bool {
	mc, ok := c.manaCost(sourceID, controller, g)
	return ok && g.CanAfford(controller, g.modifyActivationManaCost(sourceID, mc), nil)
}
func (c *permanentManaCostPayment) Pay(sourceID, controller uuid.UUID, g *Game) error {
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: sourceID, Controller: controller, Costs: []Cost{c}, ApplyActivationReduction: true})
	if err != nil {
		return err
	}
	return tx.Commit()
}

func lockPermanentManaCosts(costs []Cost, sourceID, controller uuid.UUID, g *Game) ([]Cost, error) {
	locked := costs
	copied := false
	for i, cost := range locked {
		if payment, ok := cost.(*permanentManaCostPayment); ok {
			if !copied {
				locked = append([]Cost(nil), costs...)
				copied = true
			}
			mc, present := payment.manaCost(sourceID, controller, g)
			if !present {
				return nil, fmt.Errorf("cannot pay absent permanent mana cost")
			}
			locked[i] = &ManaCostPayment{MC: mc}
		}
	}
	return locked, nil
}
