package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithLayeredLandTapTextRevision permits layer 3 changes to a printed opponent land-tap event.
// Pending triggers keep their text and event facts from detection.
func (t *GenericTriggered) WithLayeredLandTapTextRevision() *GenericTriggered {
	t.WithLandTapTextRevision()
	t.landTapTextLayer = true
	return t
}

func layeredLandTapEligible(a Ability) bool {
	rule, ok := a.(*GenericTriggered)
	return ok && rule.landTapTextLayer && rule.landTapTextRevision && rule.supportsLandTapTextRevision()
}

type printedLandTapTextEffect struct {
	effectSource
	subject   capturedPermanent
	abilityID uuid.UUID
	duration  core.Duration
	changes   []TextChange
}

// AddPrintedLandTapTextEffect changes one printed opponent land-tap declaration in layer 3.
// Each pass uses its saved baseline so expiry removes earlier changes.
func (g *Game) AddPrintedLandTapTextEffect(source *Permanent, abilityID uuid.UUID, duration core.Duration, changes ...TextChange) error {
	if source == nil || (duration != core.Indefinite && duration != core.EndOfTurn) || len(changes) == 0 {
		return fmt.Errorf("opponent land-tap text layer requires a source, changes and indefinite or end-of-turn duration")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("opponent land-tap text layer requires the original face-up source")
	}
	if !slices.ContainsFunc(current.RuntimeAbilities, func(a Ability) bool { return a.AbilityID() == abilityID && layeredLandTapEligible(a) }) {
		return fmt.Errorf("opponent land-tap text layer requires an opted-in printed declaration")
	}
	for _, change := range changes {
		if change.kind != 1 && change.kind != 2 {
			return fmt.Errorf("opponent land-tap text layer requires validated color or basic land words")
		}
	}
	probe := g.Clone()
	if err := probe.revisePrintedLandTapLayer(probe.FindPermanent(source.ID()), abilityID, changes...); err != nil {
		return err
	}
	effect := &printedLandTapTextEffect{subject: capturedPermanent{id: current.ID(), incarnation: current.incarnationID}, abilityID: abilityID, duration: duration, changes: slices.Clone(changes)}
	effect.SetSourceID(current.ID())
	g.AddContinuousEffect(effect)
	return nil
}

func (*printedLandTapTextEffect) Text() string                 { return "change printed opponent land-tap text" }
func (*printedLandTapTextEffect) Properties() EffectProperties { return EffectProperties{} }
func (*printedLandTapTextEffect) GetLayer() core.Layer         { return core.LayerText }
func (e *printedLandTapTextEffect) GetDuration() core.Duration { return e.duration }
func (e *printedLandTapTextEffect) IsActive(g *Game) bool {
	p := g.FindPermanent(e.subject.id)
	return p != nil && p.incarnationID == e.subject.incarnation && !p.FaceDown && slices.ContainsFunc(p.RuntimeAbilities, func(a Ability) bool { return a.AbilityID() == e.abilityID && layeredLandTapEligible(a) })
}
func (e *printedLandTapTextEffect) ApplyContinuous(ctx *EffectContext) error {
	g := ctx.Game
	if !e.IsActive(g) {
		return nil
	}
	p := g.MutablePermanent(e.subject.id)
	if _, exists := p.layeredLandTapOrigins[e.abilityID]; !exists {
		p.layeredLandTapOrigins = maps.Clone(p.layeredLandTapOrigins)
		if p.layeredLandTapOrigins == nil {
			p.layeredLandTapOrigins = make(map[uuid.UUID]*GenericTriggered)
		}
		for _, a := range p.RuntimeAbilities {
			if a.AbilityID() != e.abilityID {
				continue
			}
			p.layeredLandTapOrigins[e.abilityID] = a.(*GenericTriggered)
		}
	}
	return g.revisePrintedLandTapLayer(p, e.abilityID, e.changes...)
}
func (e *printedLandTapTextEffect) cloneEffect() ContinuousEffect {
	clone := *e
	clone.changes = slices.Clone(e.changes)
	return &clone
}

func (g *Game) resetPrintedLandTapTextLayers() {
	for _, p := range g.zones.battlefield {
		if p.PhasedOut || len(p.layeredLandTapOrigins) == 0 {
			continue
		}
		p = g.MutablePermanent(p.ID())
		p.RuntimeAbilities = slices.Clone(p.RuntimeAbilities)
		p.revisedPrintedTriggers = maps.Clone(p.revisedPrintedTriggers)
		for id, origin := range p.layeredLandTapOrigins {
			fresh := *origin
			fresh.source, fresh.controller = p.ID(), p.ControllerID()
			var restored Ability = &fresh
			p.revisedPrintedTriggers[id] = &fresh
			for i, a := range p.RuntimeAbilities {
				if a.AbilityID() == id {
					p.RuntimeAbilities[i] = restored
				}
			}
		}
		p.layeredLandTapOrigins = nil
	}
}

// A changed context needs a separate declaration because game clones share it.
func (g *Game) isolatePrintedLandTapContext(p *Permanent) *Permanent {
	owned := false
	for i, a := range p.RuntimeAbilities {
		if !layeredLandTapEligible(a) || (a.Source() == p.ID() && a.Controller() == p.ControllerID()) {
			continue
		}
		if !owned {
			if current := g.FindPermanentIncludingPhased(p.ID()); current != nil && current.incarnationID == p.incarnationID {
				p = g.mutablePermanentIncludingPhased(p.ID())
			}
			p.RuntimeAbilities = slices.Clone(p.RuntimeAbilities)
			owned = true
		}
		fresh := *a.(*GenericTriggered)
		fresh.source, fresh.controller = p.ID(), p.ControllerID()
		p.RuntimeAbilities[i] = &fresh
	}
	return p
}

func (g *Game) revisePrintedLandTapLayer(source *Permanent, id uuid.UUID, changes ...TextChange) error {
	var original opponentLandTappedPattern
	if source != nil {
		if current := g.FindPermanent(source.ID()); current != nil {
			for _, a := range current.RuntimeAbilities {
				if a.AbilityID() == id && layeredLandTapEligible(a) {
					original = a.(*GenericTriggered).eventDefinition.(opponentLandTappedPattern)
				}
			}
		}
	}
	return g.revisePrintedTrigger(source, id, func(t *GenericTriggered) bool {
		if !layeredLandTapEligible(t) {
			return false
		}
		pattern := t.eventDefinition.(opponentLandTappedPattern)
		return sameLandTapTextContext(original, pattern)
	}, changes...)
}

func sameLandTapTextContext(a, b opponentLandTappedPattern) bool {
	a.subtype, b.subtype = "", ""
	a.land, b.land = nil, nil
	return a == b
}
