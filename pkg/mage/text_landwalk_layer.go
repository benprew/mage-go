package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type landwalkLayerOrigin struct {
	ability    *PrintedBasicLandwalkAbility
	wasRevised bool
}

// WithLayeredPrintedBasicLandwalk permits checked layer 3 changes to one declaration.
// Granted abilities must use separate services for the ability layer.
func WithLayeredPrintedBasicLandwalk(keyword core.Attr) CardOption {
	return func(card *BaseCard) {
		WithPrintedBasicLandwalk(keyword)(card)
		card.abilities[len(card.abilities)-1].(*PrintedBasicLandwalkAbility).textLayerEligible = true
	}
}

type printedLandwalkTextEffect struct {
	effectSource
	subject   capturedPermanent
	abilityID uuid.UUID
	duration  core.Duration
	changes   []TextChange
}

// AddPrintedBasicLandwalkTextEffect changes one printed declaration in layer 3.
// Each pass uses its baseline so expiry cannot leave a later change in place.
func (g *Game) AddPrintedBasicLandwalkTextEffect(source *Permanent, abilityID uuid.UUID, duration core.Duration, changes ...TextChange) error {
	if source == nil || (duration != core.Indefinite && duration != core.EndOfTurn) || len(changes) == 0 {
		return fmt.Errorf("landwalk text layer requires a source, changes and indefinite or end-of-turn duration")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("landwalk text layer requires the original face-up source")
	}
	eligible := slices.ContainsFunc(current.RuntimeAbilities, func(a Ability) bool {
		rule, ok := a.(*PrintedBasicLandwalkAbility)
		return ok && rule.AbilityID() == abilityID && rule.textLayerEligible
	})
	if !eligible {
		return fmt.Errorf("landwalk text layer requires an opted-in printed declaration")
	}
	for _, change := range changes {
		if change.kind != 2 {
			return fmt.Errorf("landwalk text layer requires validated basic land words")
		}
	}
	probe := g.Clone()
	if err := probe.revisePrintedBasicLandwalk(probe.FindPermanent(source.ID()), abilityID, false, changes...); err != nil {
		return err
	}
	effect := &printedLandwalkTextEffect{
		subject:   capturedPermanent{id: current.ID(), incarnation: current.incarnationID},
		abilityID: abilityID, duration: duration, changes: slices.Clone(changes),
	}
	effect.SetSourceID(source.ID())
	g.AddContinuousEffect(effect)
	return nil
}

func (*printedLandwalkTextEffect) Text() string                 { return "change printed basic landwalk text" }
func (*printedLandwalkTextEffect) Properties() EffectProperties { return EffectProperties{} }
func (*printedLandwalkTextEffect) GetLayer() core.Layer         { return core.LayerText }
func (e *printedLandwalkTextEffect) GetDuration() core.Duration { return e.duration }
func (e *printedLandwalkTextEffect) IsActive(g *Game) bool {
	p := g.FindPermanent(e.subject.id)
	return p != nil && p.incarnationID == e.subject.incarnation && !p.FaceDown && slices.ContainsFunc(p.RuntimeAbilities, func(a Ability) bool {
		rule, ok := a.(*PrintedBasicLandwalkAbility)
		return ok && rule.AbilityID() == e.abilityID && rule.textLayerEligible
	})
}
func (e *printedLandwalkTextEffect) ApplyContinuous(ctx *EffectContext) error {
	g := ctx.Game
	if !e.IsActive(g) {
		return nil
	}
	p := g.MutablePermanent(e.subject.id)
	if _, exists := p.layeredLandwalkOrigins[e.abilityID]; !exists {
		p.layeredLandwalkOrigins = maps.Clone(p.layeredLandwalkOrigins)
		if p.layeredLandwalkOrigins == nil {
			p.layeredLandwalkOrigins = make(map[uuid.UUID]landwalkLayerOrigin)
		}
		for _, a := range p.RuntimeAbilities {
			if rule, ok := a.(*PrintedBasicLandwalkAbility); ok && rule.AbilityID() == e.abilityID {
				p.layeredLandwalkOrigins[e.abilityID] = landwalkLayerOrigin{rule, p.revisedPrintedLandwalks[e.abilityID] != nil}
			}
		}
	}
	return g.revisePrintedBasicLandwalk(p, e.abilityID, false, e.changes...)
}
func (e *printedLandwalkTextEffect) cloneEffect() ContinuousEffect {
	clone := *e
	clone.changes = slices.Clone(e.changes)
	return &clone
}

func (g *Game) resetPrintedLandwalkTextLayers() {
	for _, p := range g.zones.battlefield {
		if p.PhasedOut || len(p.layeredLandwalkOrigins) == 0 {
			continue
		}
		p = g.MutablePermanent(p.ID())
		p.RuntimeAbilities = slices.Clone(p.RuntimeAbilities)
		p.revisedPrintedLandwalks = maps.Clone(p.revisedPrintedLandwalks)
		for id, origin := range p.layeredLandwalkOrigins {
			revised := p.revisedPrintedLandwalks[id]
			if revised == nil {
				continue
			}
			p.baseAttrs[revised.keyword]--
			p.printedAbilityAttrs[revised.keyword]--
			restored := *origin.ability
			restored.source = p.ID()
			restored.controller = p.ControllerID()
			p.baseAttrs[restored.keyword]++
			p.printedAbilityAttrs[restored.keyword]++
			restoredActive := false
			for i, a := range p.RuntimeAbilities {
				if a.AbilityID() == id {
					p.RuntimeAbilities[i] = &restored
					restoredActive = true
				}
			}
			if !origin.wasRevised && restoredActive {
				delete(p.revisedPrintedLandwalks, id)
			} else {
				p.revisedPrintedLandwalks[id] = &restored
			}
		}
		p.layeredLandwalkOrigins = nil
	}
}

// A changed context needs a separate declaration because game clones share it.
func (g *Game) isolatePrintedLandwalkContext(p *Permanent) *Permanent {
	owned := false
	for i, ability := range p.RuntimeAbilities {
		rule, ok := ability.(*PrintedBasicLandwalkAbility)
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
