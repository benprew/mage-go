package mage

import (
	"fmt"

	"github.com/google/uuid"
)

type exactPermanentTapCost struct{ subject capturedPermanent }

func (*exactPermanentTapCost) Text() string { return "Tap the declared permanent" }
func (c *exactPermanentTapCost) CanPay(_, _ uuid.UUID, g *Game) bool {
	p := g.FindPermanent(c.subject.id)
	return p != nil && !p.PhasedOut && !p.Tapped && p.incarnationID == c.subject.incarnation
}
func (c *exactPermanentTapCost) Pay(source, controller uuid.UUID, g *Game) error {
	if !c.CanPay(source, controller, g) {
		return fmt.Errorf("cannot tap the declared permanent")
	}
	g.TapPermanent(g.FindPermanent(c.subject.id))
	return nil
}

type requiredTapPayment struct {
	player       PlayerRef
	subject      PermanentRef
	paid, unpaid ResolvingEffect
}

// RequiredTapOrElse makes a mandatory tap payment and selects one branch.
// The subject need not be controlled by the payer. The declaration supplies that rule.
func RequiredTapOrElse(player PlayerRef, subject PermanentRef, paid, unpaid ResolvingEffect) ResolvingEffect {
	return &requiredTapPayment{player: player, subject: subject, paid: paid, unpaid: unpaid}
}
func (*requiredTapPayment) Text() string { return "mandatory permanent tap payment" }
func (e *requiredTapPayment) Properties() EffectProperties {
	var props EffectProperties
	for _, body := range []ResolvingEffect{e.paid, e.unpaid} {
		if !nilDefinition(body) {
			mergeEffectProperties(&props, body.Properties())
		}
	}
	return props
}
func (e *requiredTapPayment) inspectDefinition() definitionNode {
	n := definitionNode{Role: "required-tap-payment", Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"subject", e.subject}}}
	if !nilDefinition(e.paid) {
		n.children = append(n.children, definitionEdge{"paid", e.paid})
	}
	if !nilDefinition(e.unpaid) {
		n.children = append(n.children, definitionEdge{"unpaid", e.unpaid})
	}
	if nilDefinition(e.paid) && nilDefinition(e.unpaid) {
		n.problem = "supply at least one tap payment branch"
	}
	return n
}
func (e *requiredTapPayment) Apply(ctx *EffectContext) error {
	cost := &exactPermanentTapCost{}
	if p, ok := e.subject.resolvePermanent(ctx); ok {
		cost.subject = capturedPermanent{id: p.ID(), incarnation: p.incarnationID}
	}
	paid, unpaid := e.paid, e.unpaid
	if nilDefinition(paid) {
		paid = Sequence()
	}
	if nilDefinition(unpaid) {
		unpaid = Sequence()
	}
	payment := &unlessPaidEffect{player: e.player, costs: Costs(cost), effect: paid, otherwise: unpaid, ifPaid: true, twoBranches: true, required: true}
	return payment.Apply(ctx)
}
func (e *requiredTapPayment) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject) || playerRefReadsSourcePermanent(e.player) || effectReadsSourcePermanent(e.paid) || effectReadsSourcePermanent(e.unpaid)
}
