package mage

import "github.com/google/uuid"

// PhaseOutUntilSourceLeaves keeps the subject phased out until the source leaves.
// tapOnReturn changes status during phase-in without a tap event.
func PhaseOutUntilSourceLeaves(subject PermanentRef, tapOnReturn bool) ResolvingEffect {
	return &phaseUntilEffect{subject: subject, tapOnReturn: tapOnReturn}
}

type phaseUntilEffect struct {
	subject     PermanentRef
	tapOnReturn bool
}

func (*phaseUntilEffect) Text() string                  { return "phase out until the source leaves" }
func (*phaseUntilEffect) Properties() EffectProperties  { return EffectProperties{} }
func (*phaseUntilEffect) refersToSourcePermanent() bool { return true }
func (e *phaseUntilEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), battlefieldSource: true, children: []definitionEdge{{"subject", e.subject}}}
}
func (e *phaseUntilEffect) Apply(ctx *EffectContext) error {
	source := ctx.Game.FindPermanentIncludingPhased(ctx.SourceID)
	if source == nil || (ctx.sourceIncarnation != uuid.Nil && source.incarnationID != ctx.sourceIncarnation) {
		return nil
	}
	subject, ok := e.subject.resolvePermanent(ctx)
	if !ok || subject.PhasedOut {
		return nil
	}
	sourceVisit := capturedPermanent{source.ID(), source.incarnationID}
	subjectVisit := capturedPermanent{subject.ID(), subject.incarnationID}
	ctx.Game.PhaseOut(subject.ID())
	phased := ctx.Game.FindPermanentIncludingPhased(subject.ID())
	if phased == nil || !phased.PhasedOut {
		return nil
	}
	ctx.Game.zones.phaseUntil = append(ctx.Game.zones.phaseUntil, phaseUntilWait{source: sourceVisit, subject: subjectVisit, epoch: phased.phaseEpoch, tapOnReturn: e.tapOnReturn})
	return nil
}

type phaseUntilWait struct {
	source, subject capturedPermanent
	epoch           uint64
	tapOnReturn     bool
}

func (w phaseUntilWait) subjectPresent(g *Game) *Permanent {
	p := g.FindPermanentIncludingPhased(w.subject.id)
	if p == nil || p.incarnationID != w.subject.incarnation || !p.PhasedOut || p.phaseEpoch != w.epoch {
		return nil
	}
	return p
}

func (g *Game) phaseInHeld(id uuid.UUID) bool {
	for _, w := range g.zones.phaseUntil {
		if w.subject.id == id && w.subjectPresent(g) != nil {
			return true
		}
	}
	return false
}

func (g *Game) finishPhaseUntilWaits() {
	var remaining []phaseUntilWait
	var returning []phaseUntilWait
	for _, w := range g.zones.phaseUntil {
		if w.subjectPresent(g) == nil {
			continue
		}
		source := g.FindPermanentIncludingPhased(w.source.id)
		if source != nil && source.incarnationID == w.source.incarnation {
			remaining = append(remaining, w)
		} else {
			returning = append(returning, w)
		}
	}
	g.zones.phaseUntil = remaining
	// All source moves must finish before the return group becomes present.
	for _, w := range returning {
		if w.tapOnReturn {
			if p := g.MutablePermanentIncludingPhased(w.subject.id); p != nil {
				p.Tapped = true
			}
		}
	}
	ids := make([]uuid.UUID, 0, len(returning))
	for _, w := range returning {
		ids = append(ids, w.subject.id)
	}
	g.PhaseIn(ids)
}
