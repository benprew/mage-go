package mage

import . "github.com/benprew/mage-go/pkg/mage/core"

// WhenReceivesDamageThisTurn saves each positive damage amount for one visit.
// The creator controls the delayed ability until cleanup.
func WhenReceivesDamageThisTurn(subject PermanentRef, from PermanentFilter, amount *NumberBinding, body ResolvingEffect) ResolvingEffect {
	return &delayedDamageEffect{subject: subject, from: from, amount: amount, body: body}
}

type delayedDamageEffect struct {
	subject PermanentRef
	from    PermanentFilter
	amount  *NumberBinding
	body    ResolvingEffect
}

type delayedDamageMatcher struct {
	recipient capturedPermanent
	from      PermanentFilter
	amount    *NumberBinding
}

func (*delayedDamageEffect) Text() string { return "whenever that permanent receives damage this turn" }
func (e *delayedDamageEffect) Properties() EffectProperties {
	if nilDefinition(e.body) {
		return EffectProperties{}
	}
	return e.body.Properties()
}
func (e *delayedDamageEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "delayed-damage", Text: e.Text(), isolatedBindings: true,
		children: []definitionEdge{{"subject", e.subject}, {"body", Sequence(&eventAmountCapture{binding: e.amount}, e.body)}}}
	if nilDefinition(e.subject) || e.amount == nil || nilDefinition(e.body) {
		n.problem = "delayed damage: supply a subject, amount binding and body"
	}
	return n
}
func (e *delayedDamageEffect) Apply(ctx *EffectContext) error {
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok || p.PhasedOut {
		return nil
	}
	body := captureDelayedBody(ctx, e.body)
	ctx.Game.RegisterDelayedTrigger(&DelayedTrigger{
		EventType: EvtDamageDealt, SourceID: ctx.SourceID, Controller: ctx.Controller,
		Effects: []Effect{body}, sourceCaptured: true, sourceIncarnation: body.sourceIncarnation,
		Persistent: true, damageReceived: &delayedDamageMatcher{recipient: capturedPermanent{p.ID(), p.incarnationID}, from: e.from, amount: e.amount},
	})
	return nil
}
func (m *delayedDamageMatcher) matches(evt *GameEvent, g *Game) bool {
	if evt.Amount <= 0 || evt.TargetID != m.recipient.id {
		return false
	}
	p := g.FindPermanent(evt.TargetID)
	if p == nil || p.incarnationID != m.recipient.incarnation || evt.TargetIncarnation != p.incarnationID {
		return false
	}
	source := g.damageFilterSource(evt.SourceID, g.damageEventSource)
	return source != nil && (m.from.IsZero() || m.from.Match(source, g))
}

func (e *delayedDamageEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject) || effectReadsSourcePermanent(e.body)
}
