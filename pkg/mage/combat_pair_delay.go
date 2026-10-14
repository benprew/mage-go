package mage

import . "github.com/benprew/mage-go/pkg/mage/core"

// WhenCombatPairUntilEndOfCombat watches each new opponent of one saved visit.
// The creator keeps control of the subscription after source control changes.
func WhenCombatPairUntilEndOfCombat(subject PermanentRef, filter PermanentFilter, opponent *PermanentBinding, body ResolvingEffect) ResolvingEffect {
	return &combatPairDelayEffect{subject: subject, filter: filter, opponent: opponent, body: body}
}

type combatPairDelayEffect struct {
	subject  PermanentRef
	filter   PermanentFilter
	opponent *PermanentBinding
	body     ResolvingEffect
}

func (*combatPairDelayEffect) Text() string {
	return "when that creature blocks or becomes blocked this combat"
}
func (e *combatPairDelayEffect) Properties() EffectProperties {
	if nilDefinition(e.body) {
		return EffectProperties{}
	}
	return e.body.Properties()
}
func (e *combatPairDelayEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "delayed-combat-pair", Text: e.Text(), isolatedBindings: true,
		children: []definitionEdge{{"subject", e.subject}, {"body", Sequence(&eventPermanentCapture{binding: e.opponent}, e.body)}}}
	if nilDefinition(e.subject) || e.opponent == nil || nilDefinition(e.body) {
		n.problem = "delayed combat pair: supply a subject, opponent binding and body"
	}
	return n
}
func (e *combatPairDelayEffect) Apply(ctx *EffectContext) error {
	subject, ok := e.subject.resolvePermanent(ctx)
	if !ok || subject.PhasedOut {
		return nil
	}
	body := captureDelayedBody(ctx, e.body)
	pattern := BlocksOrBecomesBlockedByCreatureMatching(This, e.filter).WithOpponent(e.opponent)
	ctx.Game.RegisterDelayedTrigger(&DelayedTrigger{
		EventType: EvtCombatPair, MatchEventID: subject.ID(), matchIncarnation: subject.incarnationID,
		SourceID: ctx.SourceID, Controller: ctx.Controller, Effects: []Effect{body},
		sourceCaptured: true, sourceIncarnation: body.sourceIncarnation,
		combatPair: &pattern, Persistent: true, expiresAtCombatEnd: true,
	})
	return nil
}
