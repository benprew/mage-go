package mage

import (
	"maps"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type delayedDeathEffect struct {
	subject PermanentRef
	body    ResolvingEffect
}

// WhenDiesThisTurn watches one battlefield object until this turn ends.
func WhenDiesThisTurn(subject PermanentRef, body ResolvingEffect) ResolvingEffect {
	return &delayedDeathEffect{subject: subject, body: body}
}

func (e *delayedDeathEffect) Text() string { return "when that permanent dies this turn" }

func (e *delayedDeathEffect) Properties() EffectProperties {
	if nilDefinition(e.body) {
		return EffectProperties{}
	}
	return e.body.Properties()
}

func (e *delayedDeathEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "delayed-event", Text: e.Text(), isolatedBindings: true,
		children: []definitionEdge{{"subject", e.subject}, {"body", e.body}}}
	if nilDefinition(e.subject) || nilDefinition(e.body) {
		n.problem = "delayed death: supply a subject and body"
	}
	return n
}

func (e *delayedDeathEffect) Apply(ctx *EffectContext) error {
	permanent, ok := e.subject.resolvePermanent(ctx)
	if !ok || permanent.PhasedOut {
		return nil
	}
	body := &capturedDelayedBody{
		body:              e.body,
		permanents:        maps.Clone(ctx.Game.resolution.permanentBindings),
		createdTokens:     cloneCreatedTokenBindings(ctx.Game.resolution.createdTokenBindings),
		players:           maps.Clone(ctx.Game.resolution.playerBindings),
		numbers:           maps.Clone(ctx.Game.resolution.numberBindings),
		targets:           cloneDelayedTargets(ctx.targetBindings),
		sourceIncarnation: ctx.sourceIncarnation,
		x:                 ctx.Game.resolution.X(),
	}
	if !ctx.sourceCaptured {
		if source := ctx.Game.FindPermanentIncludingPhased(ctx.SourceID); source != nil {
			body.sourceIncarnation = source.incarnationID
		}
	}
	ctx.Game.RegisterDelayedTrigger(&DelayedTrigger{
		EventType:         EvtZoneChange,
		MatchEventID:      permanent.ID(),
		MatchFromZone:     ZoneBattlefield,
		MatchToZone:       ZoneGraveyard,
		SourceID:          ctx.SourceID,
		Controller:        ctx.Controller,
		Effects:           []Effect{body},
		sourceCaptured:    true,
		sourceIncarnation: body.sourceIncarnation,
		matchIncarnation:  permanent.incarnationID,
		Persistent:        true,
	})
	return nil
}
