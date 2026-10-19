package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// WhenAnyLeaves watches the next departure from a saved set of present visits.
// One departure consumes the whole subscription, including simultaneous moves.
func WhenAnyLeaves(subjects PermanentSet, body ResolvingEffect) ResolvingEffect {
	return &delayedSetLeaveEffect{subjects: subjects, body: body}
}

type delayedSetLeaveEffect struct {
	subjects PermanentSet
	body     ResolvingEffect
}

func (*delayedSetLeaveEffect) Text() string {
	return "when one of those permanents leaves the battlefield"
}
func (e *delayedSetLeaveEffect) Properties() EffectProperties {
	if nilDefinition(e.body) {
		return EffectProperties{}
	}
	return e.body.Properties()
}
func (e *delayedSetLeaveEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "delayed-event", Text: e.Text(), isolatedBindings: true, children: []definitionEdge{{"subjects", e.subjects}, {"body", e.body}}}
}
func (e *delayedSetLeaveEffect) refersToSourcePermanent() bool {
	return permanentSetReadsSourcePermanent(e.subjects) || effectReadsSourcePermanent(e.body)
}
func (e *delayedSetLeaveEffect) Apply(ctx *EffectContext) error {
	var visits []capturedPermanent
	for _, p := range e.subjects.resolvePermanents(ctx) {
		if p != nil && !p.PhasedOut {
			visits = append(visits, capturedPermanent{p.ID(), p.incarnationID})
		}
	}
	if len(visits) == 0 {
		return nil
	}
	body := captureDelayedBody(ctx, e.body)
	ctx.Game.RegisterDelayedTrigger(&DelayedTrigger{
		EventType:         core.EvtZoneChange,
		MatchFromZone:     core.ZoneBattlefield,
		SourceID:          ctx.SourceID,
		Controller:        ctx.Controller,
		Effects:           []Effect{body},
		sourceCaptured:    true,
		sourceIncarnation: body.sourceIncarnation,
		leaveVisits:       visits,
	})
	return nil
}
