package mage

import (
	"fmt"

	"github.com/google/uuid"
)

type anteCost struct {
	top       bool
	locked    bool
	card      uuid.UUID
	permanent capturedPermanent
}

// AnteTopCost pays with the payer's current library top.
func AnteTopCost() Cost { return &anteCost{top: true} }

func (c *anteCost) Text() string { return "Ante a card" }
func (c *anteCost) inspectDefinition() definitionNode {
	return definitionNode{Role: "cost", Text: c.Text()}
}

func (c *anteCost) lock(controller uuid.UUID, g *Game) *anteCost {
	locked := *c
	if c.top && !c.locked {
		locked.locked = true
		if p := g.GetPlayer(controller); p != nil && len(p.Library()) > 0 {
			locked.card = p.Library()[0].ID()
		}
	}
	return &locked
}

func (c *anteCost) CanPay(_, controller uuid.UUID, g *Game) bool {
	if !g.anteEnabled || g.anteSettled {
		return false
	}
	p := g.GetPlayer(controller)
	if p == nil {
		return false
	}
	if c.top {
		if len(p.Library()) == 0 {
			return false
		}
		card := p.Library()[0]
		return card.Owner() == controller && !card.IsToken() && (!c.locked || card.ID() == c.card)
	}
	subject := g.FindPermanent(c.permanent.id)
	return subject != nil && !subject.PhasedOut && subject.incarnationID == c.permanent.incarnation && subject.Card.Owner() == controller && !subject.IsToken
}

func (c *anteCost) Pay(source, controller uuid.UUID, g *Game) error {
	if !c.CanPay(source, controller, g) {
		return fmt.Errorf("cannot pay ante cost")
	}
	if c.top {
		g.AnteLibraryTops([]uuid.UUID{controller})
		return nil
	}
	return g.MoveToAnte(controller, c.permanent.id)
}

type permanentAntePayment struct {
	player   PlayerRef
	subject  PermanentRef
	paid     ResolvingEffect
	unpaid   ResolvingEffect
	required bool
}

// PayPermanentAnteOrElse offers payment with one exact permanent.
func PayPermanentAnteOrElse(player PlayerRef, subject PermanentRef, paid, unpaid ResolvingEffect) ResolvingEffect {
	return &permanentAntePayment{player: player, subject: subject, paid: paid, unpaid: unpaid}
}

// AntePermanentOrElse pays with one exact permanent without an optional choice.
func AntePermanentOrElse(player PlayerRef, subject PermanentRef, paid, unpaid ResolvingEffect) ResolvingEffect {
	return &permanentAntePayment{player: player, subject: subject, paid: paid, unpaid: unpaid, required: true}
}

func (*permanentAntePayment) Text() string { return "ante the permanent, then resolve one branch" }
func (e *permanentAntePayment) Properties() EffectProperties {
	var props EffectProperties
	if !nilDefinition(e.paid) {
		mergeEffectProperties(&props, e.paid.Properties())
	}
	if !nilDefinition(e.unpaid) {
		mergeEffectProperties(&props, e.unpaid.Properties())
	}
	return props
}

func (e *permanentAntePayment) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-ante-payment", Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"subject", e.subject}}}
	if !nilDefinition(e.paid) {
		n.children = append(n.children, definitionEdge{"paid", e.paid})
	}
	if !nilDefinition(e.unpaid) {
		n.children = append(n.children, definitionEdge{"unpaid", e.unpaid})
	}
	if nilDefinition(e.paid) && nilDefinition(e.unpaid) {
		n.problem = "supply at least one ante payment branch"
	}
	return n
}

func (e *permanentAntePayment) Apply(ctx *EffectContext) error {
	cost := &anteCost{locked: true}
	if p, ok := e.subject.resolvePermanent(ctx); ok {
		cost.permanent = capturedPermanent{id: p.ID(), incarnation: p.incarnationID}
	}
	paid, unpaid := e.paid, e.unpaid
	if nilDefinition(paid) {
		paid = Sequence()
	}
	if nilDefinition(unpaid) {
		unpaid = Sequence()
	}
	payment := &unlessPaidEffect{player: e.player, costs: Costs(cost), effect: paid, otherwise: unpaid, ifPaid: true, twoBranches: true, required: e.required}
	return payment.Apply(ctx)
}

func (e *permanentAntePayment) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject) || playerRefReadsSourcePermanent(e.player) || effectReadsSourcePermanent(e.paid) || effectReadsSourcePermanent(e.unpaid)
}
