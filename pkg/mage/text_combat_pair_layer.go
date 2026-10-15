package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithLayeredCombatPairTextRevision permits layer 3 changes to a printed combat-pair event.
// Pending triggers keep their text and event facts from detection.
func (t *GenericTriggered) WithLayeredCombatPairTextRevision() *GenericTriggered {
	t.WithCombatPairTextRevision()
	t.combatPairTextLayer = true
	return t
}

func layeredCombatPairEligible(a Ability) bool {
	rule, ok := a.(*GenericTriggered)
	return ok && rule.combatPairTextLayer && rule.combatPairTextRevision && rule.supportsCombatPairTextRevision()
}

type printedCombatPairTextEffect struct {
	effectSource
	subject   capturedPermanent
	abilityID uuid.UUID
	duration  core.Duration
	changes   []TextChange
}

// AddPrintedCombatPairTextEffect changes one printed combat-pair declaration in layer 3.
// Each pass uses its saved baseline so expiry removes earlier changes.
func (g *Game) AddPrintedCombatPairTextEffect(source *Permanent, abilityID uuid.UUID, duration core.Duration, changes ...TextChange) error {
	if source == nil || (duration != core.Indefinite && duration != core.EndOfTurn) || len(changes) == 0 {
		return fmt.Errorf("combat-pair text layer requires a source, changes and indefinite or end-of-turn duration")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("combat-pair text layer requires the original face-up source")
	}
	if !slices.ContainsFunc(current.RuntimeAbilities, func(a Ability) bool { return a.AbilityID() == abilityID && layeredCombatPairEligible(a) }) {
		return fmt.Errorf("combat-pair text layer requires an opted-in printed declaration")
	}
	for _, change := range changes {
		if change.kind != 1 && change.kind != 2 {
			return fmt.Errorf("combat-pair text layer requires validated color or basic land words")
		}
	}
	probe := g.Clone()
	if err := probe.revisePrintedCombatPairLayer(probe.FindPermanent(source.ID()), abilityID, changes...); err != nil {
		return err
	}
	effect := &printedCombatPairTextEffect{subject: capturedPermanent{id: current.ID(), incarnation: current.incarnationID}, abilityID: abilityID, duration: duration, changes: slices.Clone(changes)}
	effect.SetSourceID(current.ID())
	if err := g.prepareAddedResolvingCopyText(current, effect); err != nil {
		return err
	}
	g.AddContinuousEffect(effect)
	return nil
}

func (*printedCombatPairTextEffect) Text() string                 { return "change printed combat-pair text" }
func (*printedCombatPairTextEffect) Properties() EffectProperties { return EffectProperties{} }
func (*printedCombatPairTextEffect) GetLayer() core.Layer         { return core.LayerText }
func (e *printedCombatPairTextEffect) GetDuration() core.Duration { return e.duration }
func (e *printedCombatPairTextEffect) IsActive(g *Game) bool {
	p := g.FindPermanent(e.subject.id)
	return p != nil && p.incarnationID == e.subject.incarnation && !p.FaceDown && slices.ContainsFunc(p.RuntimeAbilities, func(a Ability) bool { return a.AbilityID() == e.abilityID && layeredCombatPairEligible(a) })
}
func (e *printedCombatPairTextEffect) ApplyContinuous(ctx *EffectContext) error {
	g := ctx.Game
	if !e.IsActive(g) {
		return nil
	}
	p := g.MutablePermanent(e.subject.id)
	if _, exists := p.layeredCombatPairOrigins[e.abilityID]; !exists {
		p.layeredCombatPairOrigins = maps.Clone(p.layeredCombatPairOrigins)
		if p.layeredCombatPairOrigins == nil {
			p.layeredCombatPairOrigins = make(map[uuid.UUID]*GenericTriggered)
		}
		for _, a := range p.RuntimeAbilities {
			if a.AbilityID() != e.abilityID {
				continue
			}
			p.layeredCombatPairOrigins[e.abilityID] = a.(*GenericTriggered)
		}
	}
	return g.revisePrintedCombatPairLayer(p, e.abilityID, e.changes...)
}
func (e *printedCombatPairTextEffect) cloneEffect() ContinuousEffect {
	clone := *e
	clone.changes = slices.Clone(e.changes)
	return &clone
}

func (g *Game) resetPrintedCombatPairTextLayers() {
	for _, p := range g.zones.battlefield {
		if p.PhasedOut || len(p.layeredCombatPairOrigins) == 0 {
			continue
		}
		p = g.MutablePermanent(p.ID())
		p.RuntimeAbilities = slices.Clone(p.RuntimeAbilities)
		p.revisedPrintedTriggers = maps.Clone(p.revisedPrintedTriggers)
		for id, origin := range p.layeredCombatPairOrigins {
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
		p.layeredCombatPairOrigins = nil
	}
}

// A changed context needs a separate declaration because game clones share it.
func (g *Game) isolatePrintedCombatPairContext(p *Permanent) *Permanent {
	owned := false
	for i, a := range p.RuntimeAbilities {
		if !layeredCombatPairEligible(a) || (a.Source() == p.ID() && a.Controller() == p.ControllerID()) {
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

func (g *Game) revisePrintedCombatPairLayer(source *Permanent, id uuid.UUID, changes ...TextChange) error {
	var original filteredCombatPairPattern
	if source != nil {
		if current := g.FindPermanent(source.ID()); current != nil {
			for _, a := range current.RuntimeAbilities {
				if a.AbilityID() == id && layeredCombatPairEligible(a) {
					original = a.(*GenericTriggered).eventDefinition.(filteredCombatPairPattern)
				}
			}
		}
	}
	return g.revisePrintedTrigger(source, id, func(t *GenericTriggered) bool {
		if !layeredCombatPairEligible(t) {
			return false
		}
		pattern := t.eventDefinition.(filteredCombatPairPattern)
		return sameCombatPairTextContext(original, pattern)
	}, changes...)
}

func sameCombatPairTextContext(a, b filteredCombatPairPattern) bool {
	sameSubject := false
	switch a.subject.(type) {
	case sourceRef:
		_, sameSubject = b.subject.(sourceRef)
	case attachedPermanentRef:
		_, sameSubject = b.subject.(attachedPermanentRef)
	}
	return sameSubject && a.kind == b.kind && a.captureOpponent == b.captureOpponent && a.duplicateCapture == b.duplicateCapture
}
