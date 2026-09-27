package mage

import (
	"maps"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type delayedDepartureEffect struct {
	subject PermanentRef
	body    ResolvingEffect
	leaves  bool
}

// WhenDiesThisTurn watches one battlefield object until this turn ends.
func WhenDiesThisTurn(subject PermanentRef, body ResolvingEffect) ResolvingEffect {
	return &delayedDepartureEffect{subject: subject, body: body}
}

// WhenLeavesThisTurn watches one battlefield object until this turn ends.
func WhenLeavesThisTurn(subject PermanentRef, body ResolvingEffect) ResolvingEffect {
	return &delayedDepartureEffect{subject: subject, body: body, leaves: true}
}

func (e *delayedDepartureEffect) Text() string {
	if e.leaves {
		return "when that permanent leaves the battlefield this turn"
	}
	return "when that permanent dies this turn"
}

func (e *delayedDepartureEffect) Properties() EffectProperties {
	if nilDefinition(e.body) {
		return EffectProperties{}
	}
	return e.body.Properties()
}

func (e *delayedDepartureEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "delayed-event", Text: e.Text(), isolatedBindings: true,
		children: []definitionEdge{{"subject", e.subject}, {"body", e.body}}}
	if nilDefinition(e.subject) || nilDefinition(e.body) {
		n.problem = "delayed departure: supply a subject and body"
	}
	return n
}

func (e *delayedDepartureEffect) Apply(ctx *EffectContext) error {
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
	dt := &DelayedTrigger{
		EventType:         EvtZoneChange,
		MatchEventID:      permanent.ID(),
		MatchFromZone:     ZoneBattlefield,
		SourceID:          ctx.SourceID,
		Controller:        ctx.Controller,
		Effects:           []Effect{body},
		sourceCaptured:    true,
		sourceIncarnation: body.sourceIncarnation,
		matchIncarnation:  permanent.incarnationID,
		Persistent:        true,
	}
	if !e.leaves {
		dt.MatchToZone = ZoneGraveyard
	}
	ctx.Game.RegisterDelayedTrigger(dt)
	return nil
}
