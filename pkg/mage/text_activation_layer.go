package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithLayeredTextRevisionActivationState permits layer 3 changes to a printed action.
func WithLayeredTextRevisionActivationState() ActionOption {
	return func(a *ActionDefinition) {
		WithTextRevisionActivationState()(a)
		a.activationTextLayer = true
	}
}

func layeredActivatedEligible(a Ability) bool {
	rule, ok := a.(*ActionDefinition)
	return ok && rule.activationTextLayer && rule.textRevisionState && rule.kind == ActionActivated && !rule.staticGrant
}

type printedActivatedTextEffect struct {
	effectSource
	subject   capturedPermanent
	abilityID uuid.UUID
	duration  core.Duration
	changes   []TextChange
}

// AddPrintedActivatedTextEffect changes one printed activated action in layer 3.
// Each pass uses its saved baseline so expiry removes earlier changes.
func (g *Game) AddPrintedActivatedTextEffect(source *Permanent, abilityID uuid.UUID, duration core.Duration, changes ...TextChange) error {
	if source == nil || (duration != core.Indefinite && duration != core.EndOfTurn) || len(changes) == 0 {
		return fmt.Errorf("activated-action text layer requires a source, changes and indefinite or end-of-turn duration")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("activated-action text layer requires the original face-up source")
	}
	if !slices.ContainsFunc(current.RuntimeAbilities, func(a Ability) bool { return a.AbilityID() == abilityID && layeredActivatedEligible(a) }) {
		return fmt.Errorf("activated-action text layer requires an opted-in printed declaration")
	}
	for _, change := range changes {
		if change.kind != 1 && change.kind != 2 {
			return fmt.Errorf("activated-action text layer requires validated color or basic land words")
		}
	}
	probe := g.Clone()
	if err := probe.revisePrintedActivatedAbility(probe.FindPermanent(source.ID()), abilityID, changes...); err != nil {
		return err
	}
	effect := &printedActivatedTextEffect{subject: capturedPermanent{id: current.ID(), incarnation: current.incarnationID}, abilityID: abilityID, duration: duration, changes: slices.Clone(changes)}
	effect.SetSourceID(current.ID())
	g.AddContinuousEffect(effect)
	return nil
}

func (*printedActivatedTextEffect) Text() string                 { return "change printed activated-action text" }
func (*printedActivatedTextEffect) Properties() EffectProperties { return EffectProperties{} }
func (*printedActivatedTextEffect) GetLayer() core.Layer         { return core.LayerText }
func (e *printedActivatedTextEffect) GetDuration() core.Duration { return e.duration }
func (e *printedActivatedTextEffect) IsActive(g *Game) bool {
	p := g.FindPermanent(e.subject.id)
	return p != nil && p.incarnationID == e.subject.incarnation && !p.FaceDown && slices.ContainsFunc(p.RuntimeAbilities, func(a Ability) bool { return a.AbilityID() == e.abilityID && layeredActivatedEligible(a) })
}
func (e *printedActivatedTextEffect) ApplyContinuous(ctx *EffectContext) error {
	g := ctx.Game
	if !e.IsActive(g) {
		return nil
	}
	p := g.MutablePermanent(e.subject.id)
	if _, exists := p.layeredActivatedOrigins[e.abilityID]; !exists {
		p.layeredActivatedOrigins = maps.Clone(p.layeredActivatedOrigins)
		if p.layeredActivatedOrigins == nil {
			p.layeredActivatedOrigins = make(map[uuid.UUID]*ActionDefinition)
		}
		for _, a := range p.RuntimeAbilities {
			if a.AbilityID() != e.abilityID {
				continue
			}
			p.layeredActivatedOrigins[e.abilityID] = a.(*ActionDefinition)
		}
	}
	return g.revisePrintedActivatedAbility(p, e.abilityID, e.changes...)
}
func (e *printedActivatedTextEffect) cloneEffect() ContinuousEffect {
	clone := *e
	clone.changes = slices.Clone(e.changes)
	return &clone
}

func (g *Game) resetPrintedActivatedTextLayers() {
	for _, p := range g.zones.battlefield {
		if p.PhasedOut || len(p.layeredActivatedOrigins) == 0 {
			continue
		}
		p = g.MutablePermanent(p.ID())
		p.RuntimeAbilities = slices.Clone(p.RuntimeAbilities)
		p.revisedPrintedActions = maps.Clone(p.revisedPrintedActions)
		for id, origin := range p.layeredActivatedOrigins {
			fresh := *origin
			fresh.source, fresh.controller = p.ID(), p.ControllerID()
			state := p.revisedPrintedActions[id]
			for _, a := range p.RuntimeAbilities {
				if a.AbilityID() == id {
					state, _ = a.(*ActionDefinition)
				}
			}
			if state != nil {
				fresh.activatedThisTurn = state.activatedThisTurn
				fresh.activationsThisTurn = state.activationsThisTurn
			}
			var restored Ability = &fresh
			p.revisedPrintedActions[id] = &fresh
			for i, a := range p.RuntimeAbilities {
				if a.AbilityID() == id {
					p.RuntimeAbilities[i] = restored
				}
			}
		}
		p.layeredActivatedOrigins = nil
	}
}

// A changed context needs a separate declaration because game clones share it.
func (g *Game) isolatePrintedActivatedContext(p *Permanent) *Permanent {
	owned := false
	for i, a := range p.RuntimeAbilities {
		if !layeredActivatedEligible(a) || a.Source() == uuid.Nil || (a.Source() == p.ID() && a.Controller() == p.ControllerID()) {
			continue
		}
		if !owned {
			if current := g.FindPermanentIncludingPhased(p.ID()); current != nil && current.incarnationID == p.incarnationID {
				p = g.mutablePermanentIncludingPhased(p.ID())
			}
			p.RuntimeAbilities = slices.Clone(p.RuntimeAbilities)
			owned = true
		}
		fresh := *a.(*ActionDefinition)
		fresh.source, fresh.controller = p.ID(), p.ControllerID()
		p.RuntimeAbilities[i] = &fresh
	}
	return p
}
