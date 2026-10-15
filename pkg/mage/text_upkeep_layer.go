package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithLayeredUpkeepTextRevision permits layer 3 changes to a printed upkeep event.
// Pending triggers keep their text and event facts from detection.
func (t *GenericTriggered) WithLayeredUpkeepTextRevision() *GenericTriggered {
	t.WithUpkeepTextRevision()
	t.upkeepTextLayer = true
	return t
}

func layeredUpkeepEligible(a Ability) bool {
	rule, ok := a.(*GenericTriggered)
	return ok && rule.upkeepTextLayer && rule.upkeepTextRevision && rule.supportsUpkeepTextRevision()
}

type printedUpkeepTextEffect struct {
	effectSource
	subject   capturedPermanent
	abilityID uuid.UUID
	duration  core.Duration
	changes   []TextChange
}

// AddPrintedUpkeepTextEffect changes one printed upkeep declaration in layer 3.
// Each pass uses its saved baseline so expiry removes earlier changes.
func (g *Game) AddPrintedUpkeepTextEffect(source *Permanent, abilityID uuid.UUID, duration core.Duration, changes ...TextChange) error {
	if source == nil || (duration != core.Indefinite && duration != core.EndOfTurn) || len(changes) == 0 {
		return fmt.Errorf("upkeep text layer requires a source, changes and indefinite or end-of-turn duration")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("upkeep text layer requires the original face-up source")
	}
	for _, ability := range current.RuntimeAbilities {
		if nilDefinition(ability) {
			return fmt.Errorf("upkeep text layer requires valid runtime declarations")
		}
	}
	if !slices.ContainsFunc(current.RuntimeAbilities, func(a Ability) bool { return a.AbilityID() == abilityID && layeredUpkeepEligible(a) }) {
		return fmt.Errorf("upkeep text layer requires an opted-in printed declaration")
	}
	for _, change := range changes {
		if change.kind != 1 && change.kind != 2 {
			return fmt.Errorf("upkeep text layer requires validated color or basic land words")
		}
	}
	probe := g.Clone()
	if err := probe.revisePrintedUpkeepLayer(probe.FindPermanent(source.ID()), abilityID, changes...); err != nil {
		return err
	}
	effect := &printedUpkeepTextEffect{subject: capturedPermanent{id: current.ID(), incarnation: current.incarnationID}, abilityID: abilityID, duration: duration, changes: slices.Clone(changes)}
	effect.SetSourceID(current.ID())
	if err := g.prepareAddedResolvingCopyText(current, effect); err != nil {
		return err
	}
	g.AddContinuousEffect(effect)
	return nil
}

func (*printedUpkeepTextEffect) Text() string                 { return "change printed upkeep text" }
func (*printedUpkeepTextEffect) Properties() EffectProperties { return EffectProperties{} }
func (*printedUpkeepTextEffect) GetLayer() core.Layer         { return core.LayerText }
func (e *printedUpkeepTextEffect) GetDuration() core.Duration { return e.duration }
func (e *printedUpkeepTextEffect) IsActive(g *Game) bool {
	p := g.FindPermanent(e.subject.id)
	return p != nil && p.incarnationID == e.subject.incarnation && !p.FaceDown && slices.ContainsFunc(p.RuntimeAbilities, func(a Ability) bool { return a.AbilityID() == e.abilityID && layeredUpkeepEligible(a) })
}
func (e *printedUpkeepTextEffect) ApplyContinuous(ctx *EffectContext) error {
	g := ctx.Game
	if !e.IsActive(g) {
		return nil
	}
	p := g.MutablePermanent(e.subject.id)
	if _, exists := p.layeredUpkeepOrigins[e.abilityID]; !exists {
		p.layeredUpkeepOrigins = maps.Clone(p.layeredUpkeepOrigins)
		if p.layeredUpkeepOrigins == nil {
			p.layeredUpkeepOrigins = make(map[uuid.UUID]*GenericTriggered)
		}
		for _, a := range p.RuntimeAbilities {
			if a.AbilityID() != e.abilityID {
				continue
			}
			p.layeredUpkeepOrigins[e.abilityID] = a.(*GenericTriggered)
		}
	}
	return g.revisePrintedUpkeepLayer(p, e.abilityID, e.changes...)
}
func (e *printedUpkeepTextEffect) cloneEffect() ContinuousEffect {
	clone := *e
	clone.changes = slices.Clone(e.changes)
	return &clone
}

func (g *Game) resetPrintedUpkeepTextLayers() {
	for _, p := range g.zones.battlefield {
		if p.PhasedOut || len(p.layeredUpkeepOrigins) == 0 {
			continue
		}
		p = g.MutablePermanent(p.ID())
		p.RuntimeAbilities = slices.Clone(p.RuntimeAbilities)
		p.revisedPrintedTriggers = maps.Clone(p.revisedPrintedTriggers)
		for id, origin := range p.layeredUpkeepOrigins {
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
		p.layeredUpkeepOrigins = nil
	}
}

// A changed context needs a separate declaration because game clones share it.
func (g *Game) isolatePrintedUpkeepContext(p *Permanent) *Permanent {
	owned := false
	for i, a := range p.RuntimeAbilities {
		if !layeredUpkeepEligible(a) || (a.Source() == p.ID() && a.Controller() == p.ControllerID()) {
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

func (g *Game) revisePrintedUpkeepLayer(source *Permanent, id uuid.UUID, changes ...TextChange) error {
	return g.revisePrintedUpkeepTrigger(source, id, true, changes...)
}
