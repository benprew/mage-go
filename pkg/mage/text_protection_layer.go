package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type protectionLayerOrigin struct {
	ability *ProtectionAbility
}

// WithLayeredColorTextRevision permits checked layer 3 changes to printed protection.
// Granted protection must use a separate ability-layer service.
func (pa *ProtectionAbility) WithLayeredColorTextRevision() *ProtectionAbility {
	pa.WithColorTextRevision()
	pa.textLayerEligible = true
	return pa
}

type printedProtectionTextEffect struct {
	effectSource
	subject   capturedPermanent
	abilityID uuid.UUID
	duration  core.Duration
	changes   []TextChange
}

// AddPrintedProtectionTextEffect changes one printed declaration in layer 3.
// Each pass uses its baseline so expiry cannot leave a later change in place.
func (g *Game) AddPrintedProtectionTextEffect(source *Permanent, abilityID uuid.UUID, duration core.Duration, changes ...TextChange) error {
	if source == nil || (duration != core.Indefinite && duration != core.EndOfTurn) || len(changes) == 0 {
		return fmt.Errorf("protection text layer requires a source, changes and indefinite or end-of-turn duration")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("protection text layer requires the original face-up source")
	}
	eligible := slices.ContainsFunc(current.RuntimeAbilities, func(a Ability) bool {
		rule, ok := a.(*ProtectionAbility)
		return ok && rule.AbilityID() == abilityID && rule.textLayerEligible && rule.supportsColorTextRevision()
	})
	if !eligible {
		return fmt.Errorf("protection text layer requires an opted-in printed declaration")
	}
	for _, change := range changes {
		if change.kind != 1 {
			return fmt.Errorf("protection text layer requires validated color words")
		}
	}
	probe := g.Clone()
	if err := probe.revisePrintedProtection(probe.FindPermanent(source.ID()), abilityID, false, changes...); err != nil {
		return err
	}
	effect := &printedProtectionTextEffect{
		subject:   capturedPermanent{id: current.ID(), incarnation: current.incarnationID},
		abilityID: abilityID, duration: duration, changes: slices.Clone(changes),
	}
	effect.SetSourceID(source.ID())
	if err := g.prepareAddedResolvingCopyText(current, effect); err != nil {
		return err
	}
	g.AddContinuousEffect(effect)
	return nil
}

func (*printedProtectionTextEffect) Text() string                 { return "change printed color protection text" }
func (*printedProtectionTextEffect) Properties() EffectProperties { return EffectProperties{} }
func (*printedProtectionTextEffect) GetLayer() core.Layer         { return core.LayerText }
func (e *printedProtectionTextEffect) GetDuration() core.Duration { return e.duration }
func (e *printedProtectionTextEffect) IsActive(g *Game) bool {
	p := g.FindPermanent(e.subject.id)
	return p != nil && p.incarnationID == e.subject.incarnation && !p.FaceDown && slices.ContainsFunc(p.RuntimeAbilities, func(a Ability) bool {
		rule, ok := a.(*ProtectionAbility)
		return ok && rule.AbilityID() == e.abilityID && rule.textLayerEligible && rule.supportsColorTextRevision()
	})
}
func (e *printedProtectionTextEffect) ApplyContinuous(ctx *EffectContext) error {
	g := ctx.Game
	if !e.IsActive(g) {
		return nil
	}
	p := g.MutablePermanent(e.subject.id)
	if _, exists := p.layeredProtectionOrigins[e.abilityID]; !exists {
		p.layeredProtectionOrigins = maps.Clone(p.layeredProtectionOrigins)
		if p.layeredProtectionOrigins == nil {
			p.layeredProtectionOrigins = make(map[uuid.UUID]protectionLayerOrigin)
		}
		for _, a := range p.RuntimeAbilities {
			if rule, ok := a.(*ProtectionAbility); ok && rule.AbilityID() == e.abilityID {
				p.layeredProtectionOrigins[e.abilityID] = protectionLayerOrigin{rule}
			}
		}
	}
	return g.revisePrintedProtection(p, e.abilityID, false, e.changes...)
}
func (e *printedProtectionTextEffect) cloneEffect() ContinuousEffect {
	clone := *e
	clone.changes = slices.Clone(e.changes)
	return &clone
}

func (g *Game) resetPrintedProtectionTextLayers() {
	for _, p := range g.zones.battlefield {
		if p.PhasedOut || len(p.layeredProtectionOrigins) == 0 {
			continue
		}
		p = g.MutablePermanent(p.ID())
		p.RuntimeAbilities = slices.Clone(p.RuntimeAbilities)
		p.revisedPrintedProtections = maps.Clone(p.revisedPrintedProtections)
		for id, origin := range p.layeredProtectionOrigins {
			revised := p.revisedPrintedProtections[id]
			if revised == nil {
				continue
			}
			restored := *origin.ability
			restored.source = p.ID()
			restored.controller = p.ControllerID()
			for i, a := range p.RuntimeAbilities {
				if a.AbilityID() == id {
					p.RuntimeAbilities[i] = &restored
				}
			}
			p.revisedPrintedProtections[id] = &restored
		}
		p.layeredProtectionOrigins = nil
	}
}

// A changed context needs a separate declaration because game clones share it.
func (g *Game) isolatePrintedProtectionContext(p *Permanent) *Permanent {
	owned := false
	for i, ability := range p.RuntimeAbilities {
		rule, ok := ability.(*ProtectionAbility)
		if !ok || !rule.textLayerEligible || (rule.Source() == p.ID() && rule.Controller() == p.ControllerID()) {
			continue
		}
		if !owned {
			if current := g.FindPermanentIncludingPhased(p.ID()); current != nil && current.incarnationID == p.incarnationID {
				p = g.mutablePermanentIncludingPhased(p.ID())
			}
			p.RuntimeAbilities = slices.Clone(p.RuntimeAbilities)
			owned = true
		}
		fresh := *rule
		fresh.source = p.ID()
		fresh.controller = p.ControllerID()
		p.RuntimeAbilities[i] = &fresh
	}
	return p
}
