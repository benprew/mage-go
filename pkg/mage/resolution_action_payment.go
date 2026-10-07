package mage

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ResolutionActionPayment is one action paid during resolution.
type ResolutionActionPayment interface {
	prepareResolutionAction(*EffectContext) (preparedResolutionAction, bool)
}

type resolutionActionKind uint8

const (
	resolutionGainLife resolutionActionKind = iota
	resolutionDamage
	resolutionDestroy
)

type resolutionActionPayment struct {
	kind      resolutionActionKind
	player    PlayerRef
	permanent PermanentRef
	source    DamageSourceRef
	recipient DamageRecipientRef
	amount    ResolutionValue
}

// GainLifePayment uses the selected player's life gain as a resolution cost.
func GainLifePayment(player PlayerRef, amount ResolutionValue) ResolutionActionPayment {
	return &resolutionActionPayment{kind: resolutionGainLife, player: player, amount: amount}
}

// DamagePayment uses one damage instruction as a resolution cost.
func DamagePayment(source DamageSourceRef, recipient DamageRecipientRef, amount ResolutionValue) ResolutionActionPayment {
	return &resolutionActionPayment{kind: resolutionDamage, source: source, recipient: recipient, amount: amount}
}

// DestroyPayment requires a live permanent that can be destroyed.
func DestroyPayment(subject PermanentRef) ResolutionActionPayment {
	return &resolutionActionPayment{kind: resolutionDestroy, permanent: subject}
}

type preparedResolutionAction struct {
	kind            resolutionActionKind
	id, incarnation uuid.UUID
	source          DamageSource
	amount          int
}

func (e *resolutionActionPayment) prepareResolutionAction(ctx *EffectContext) (preparedResolutionAction, bool) {
	prepared := preparedResolutionAction{kind: e.kind}
	var ok bool
	switch e.kind {
	case resolutionGainLife:
		prepared.id, ok = e.player.resolvePlayer(ctx)
	case resolutionDamage:
		prepared.source = e.source.resolveDamageSource(ctx)
		if !prepared.source.Valid() {
			return prepared, false
		}
		prepared.id, ok = e.recipient.resolveRecipient(ctx)
		if p := ctx.Game.FindPermanent(prepared.id); p != nil {
			prepared.incarnation = p.incarnationID
		}
	case resolutionDestroy:
		p, present := e.permanent.resolvePermanent(ctx)
		if !present || p.HasAttr(core.Indestructible) {
			return prepared, false
		}
		prepared.id, prepared.incarnation, ok = p.ID(), p.incarnationID, true
	}
	if ok && e.kind != resolutionDestroy {
		prepared.amount = max(0, e.amount.resolveValue(ctx))
	}
	return prepared, ok
}

func (p preparedResolutionAction) available(g *Game) bool {
	if p.incarnation == uuid.Nil {
		return g.GetPlayer(p.id) != nil
	}
	current := g.FindPermanent(p.id)
	return current != nil && !current.PhasedOut && current.incarnationID == p.incarnation && (p.kind != resolutionDestroy || !current.HasAttr(core.Indestructible))
}

func (p preparedResolutionAction) pay(ctx *EffectContext) {
	switch p.kind {
	case resolutionGainLife:
		if p.amount > 0 {
			ctx.Game.playersGainLife([]Player{ctx.Game.GetPlayer(p.id)}, p.amount)
		}
	case resolutionDamage:
		if p.amount <= 0 {
			return
		}
		if player := ctx.Game.GetPlayer(p.id); player != nil {
			ctx.Game.damage.dealDamageToPlayerFromSourceOutcome(ctx.Game, player, p.amount, p.source)
		} else {
			ctx.Game.damage.dealDamageToPermanentFromSourceOutcome(ctx.Game, ctx.Game.FindPermanent(p.id), p.amount, p.source)
		}
	case resolutionDestroy:
		ctx.Game.destroyPermanents([]*Permanent{ctx.Game.FindPermanent(p.id)}, false)
	}
}

func (e *resolutionActionPayment) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolution-action-payment"}
	switch e.kind {
	case resolutionGainLife:
		n.children = []definitionEdge{{"player", e.player}, {"amount", e.amount}}
	case resolutionDamage:
		n.children = []definitionEdge{{"source", damageSourceDefinition{e.source}}, {"recipient", e.recipient}, {"amount", e.amount}}
	case resolutionDestroy:
		n.children = []definitionEdge{{"subject", e.permanent}}
	}
	return n
}

type resolutionActionPaymentEffect struct {
	player       PlayerRef
	payment      ResolutionActionPayment
	paid, unpaid ResolvingEffect
}

// PayActionOrElse keeps the payment choice separate from the action's outcome.
func PayActionOrElse(player PlayerRef, payment ResolutionActionPayment, paid, unpaid ResolvingEffect) ResolvingEffect {
	return &resolutionActionPaymentEffect{player: player, payment: payment, paid: paid, unpaid: unpaid}
}
func (*resolutionActionPaymentEffect) Text() string { return "pay an action, then resolve one branch" }
func (e *resolutionActionPaymentEffect) Properties() EffectProperties {
	props := EffectProperties{}
	if !nilDefinition(e.paid) {
		props = e.paid.Properties()
	}
	if !nilDefinition(e.unpaid) {
		mergeEffectProperties(&props, e.unpaid.Properties())
	}
	if payment, ok := e.payment.(*resolutionActionPayment); ok {
		switch payment.kind {
		case resolutionGainLife:
			mergeEffectProperties(&props, GainLifeFor(payment.player, payment.amount).Properties())
		case resolutionDamage:
			mergeEffectProperties(&props, DealDamageFrom(payment.source, payment.recipient, payment.amount).Properties())
		case resolutionDestroy:
			mergeEffectProperties(&props, Destroy(payment.permanent).Properties())
		}
	}
	return props
}
func (e *resolutionActionPaymentEffect) Apply(ctx *EffectContext) error {
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	prepared, available := e.payment.prepareResolutionAction(ctx)
	paid := false
	if available && ctx.Game.GetPlayer(id).ChooseMayAbility(prepared.prompt(ctx.Game)) && prepared.available(ctx.Game) {
		// Prevention and replacement do not change an accepted payment.
		prepared.pay(ctx)
		paid = true
	}
	branch := e.unpaid
	if paid {
		branch = e.paid
	}
	if nilDefinition(branch) {
		return nil
	}
	return branch.Apply(ctx)
}

func (p preparedResolutionAction) prompt(g *Game) string {
	switch p.kind {
	case resolutionGainLife:
		return fmt.Sprintf("gain %d life", p.amount)
	case resolutionDamage:
		return fmt.Sprintf("deal %d damage", p.amount)
	default:
		return "destroy " + g.FindPermanent(p.id).Name()
	}
}
func (e *resolutionActionPaymentEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolution-action-payment", Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"payment", e.payment}}}
	if !nilDefinition(e.paid) {
		n.children = append(n.children, definitionEdge{"paid", e.paid})
	}
	if !nilDefinition(e.unpaid) {
		n.children = append(n.children, definitionEdge{"unpaid", e.unpaid})
	}
	if nilDefinition(e.paid) && nilDefinition(e.unpaid) {
		n.problem = "supply at least one action payment branch"
	}
	return n
}
