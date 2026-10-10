package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithLayeredManaTapTextRevision permits layer 3 changes to a printed mana event.
// Pending triggers keep their text and event facts from detection.
func (t *GenericTriggered) WithLayeredManaTapTextRevision() *GenericTriggered {
	t.WithManaTapTextRevision()
	t.manaTapTextLayer = true
	return t
}

// WithLayeredManaTapTextRevision permits layer 3 changes to a printed mana bonus.
// The shared mana service keeps payment planning and production together.
func (a *ManaBonusAbility) WithLayeredManaTapTextRevision() *ManaBonusAbility {
	a.WithManaTapTextRevision()
	a.manaTapTextLayer = true
	return a
}

type manaTapLayerOrigin struct {
	trigger *GenericTriggered
	bonus   *ManaBonusAbility
}

func layeredManaTapEligible(a Ability) bool {
	switch rule := a.(type) {
	case *GenericTriggered:
		return rule.manaTapTextLayer && rule.manaTapTextRevision && rule.supportsManaTapTextRevision()
	case *ManaBonusAbility:
		return rule.manaTapTextLayer && rule.manaTapTextRevision && rule.supportsManaTapTextRevision()
	}
	return false
}

type printedManaTapTextEffect struct {
	effectSource
	subject   capturedPermanent
	abilityID uuid.UUID
	duration  core.Duration
	changes   []TextChange
}

// AddPrintedManaTapTextEffect changes one printed mana-event declaration in layer 3.
// Each pass uses its saved baseline so expiry removes earlier changes.
func (g *Game) AddPrintedManaTapTextEffect(source *Permanent, abilityID uuid.UUID, duration core.Duration, changes ...TextChange) error {
	if source == nil || (duration != core.Indefinite && duration != core.EndOfTurn) || len(changes) == 0 {
		return fmt.Errorf("mana-tap text layer requires a source, changes and indefinite or end-of-turn duration")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("mana-tap text layer requires the original face-up source")
	}
	if !slices.ContainsFunc(current.RuntimeAbilities, func(a Ability) bool { return a.AbilityID() == abilityID && layeredManaTapEligible(a) }) {
		return fmt.Errorf("mana-tap text layer requires an opted-in printed declaration")
	}
	for _, change := range changes {
		if change.kind != 1 && change.kind != 2 {
			return fmt.Errorf("mana-tap text layer requires validated color or basic land words")
		}
	}
	probe := g.Clone()
	if err := probe.revisePrintedManaTapTrigger(probe.FindPermanent(source.ID()), abilityID, false, changes...); err != nil {
		return err
	}
	effect := &printedManaTapTextEffect{subject: capturedPermanent{id: current.ID(), incarnation: current.incarnationID}, abilityID: abilityID, duration: duration, changes: slices.Clone(changes)}
	effect.SetSourceID(current.ID())
	g.AddContinuousEffect(effect)
	return nil
}

func (*printedManaTapTextEffect) Text() string                 { return "change printed mana-event text" }
func (*printedManaTapTextEffect) Properties() EffectProperties { return EffectProperties{} }
func (*printedManaTapTextEffect) GetLayer() core.Layer         { return core.LayerText }
func (e *printedManaTapTextEffect) GetDuration() core.Duration { return e.duration }
func (e *printedManaTapTextEffect) IsActive(g *Game) bool {
	p := g.FindPermanent(e.subject.id)
	return p != nil && p.incarnationID == e.subject.incarnation && !p.FaceDown && slices.ContainsFunc(p.RuntimeAbilities, func(a Ability) bool { return a.AbilityID() == e.abilityID && layeredManaTapEligible(a) })
}
func (e *printedManaTapTextEffect) ApplyContinuous(ctx *EffectContext) error {
	g := ctx.Game
	if !e.IsActive(g) {
		return nil
	}
	p := g.MutablePermanent(e.subject.id)
	if _, exists := p.layeredManaTapOrigins[e.abilityID]; !exists {
		p.layeredManaTapOrigins = maps.Clone(p.layeredManaTapOrigins)
		if p.layeredManaTapOrigins == nil {
			p.layeredManaTapOrigins = make(map[uuid.UUID]manaTapLayerOrigin)
		}
		for _, a := range p.RuntimeAbilities {
			if a.AbilityID() != e.abilityID {
				continue
			}
			switch rule := a.(type) {
			case *GenericTriggered:
				p.layeredManaTapOrigins[e.abilityID] = manaTapLayerOrigin{trigger: rule}
			case *ManaBonusAbility:
				p.layeredManaTapOrigins[e.abilityID] = manaTapLayerOrigin{bonus: rule}
			}
		}
	}
	return g.revisePrintedManaTapTrigger(p, e.abilityID, false, e.changes...)
}
func (e *printedManaTapTextEffect) cloneEffect() ContinuousEffect {
	clone := *e
	clone.changes = slices.Clone(e.changes)
	return &clone
}

func (g *Game) resetPrintedManaTapTextLayers() {
	for _, p := range g.zones.battlefield {
		if p.PhasedOut || len(p.layeredManaTapOrigins) == 0 {
			continue
		}
		p = g.MutablePermanent(p.ID())
		p.RuntimeAbilities = slices.Clone(p.RuntimeAbilities)
		p.revisedPrintedTriggers = maps.Clone(p.revisedPrintedTriggers)
		p.revisedPrintedManaBonuses = maps.Clone(p.revisedPrintedManaBonuses)
		for id, origin := range p.layeredManaTapOrigins {
			var restored Ability
			if origin.trigger != nil {
				fresh := *origin.trigger
				fresh.source, fresh.controller = p.ID(), p.ControllerID()
				restored = &fresh
				p.revisedPrintedTriggers[id] = &fresh
			} else {
				fresh := *origin.bonus
				fresh.source, fresh.controller = p.ID(), p.ControllerID()
				restored = &fresh
				p.revisedPrintedManaBonuses[id] = &fresh
			}
			for i, a := range p.RuntimeAbilities {
				if a.AbilityID() == id {
					p.RuntimeAbilities[i] = restored
				}
			}
		}
		p.layeredManaTapOrigins = nil
	}
}

// A changed context needs a separate declaration because game clones share it.
func (g *Game) isolatePrintedManaTapContext(p *Permanent) *Permanent {
	owned := false
	for i, a := range p.RuntimeAbilities {
		if !layeredManaTapEligible(a) || (a.Source() == p.ID() && a.Controller() == p.ControllerID()) {
			continue
		}
		if !owned {
			if current := g.FindPermanentIncludingPhased(p.ID()); current != nil && current.incarnationID == p.incarnationID {
				p = g.mutablePermanentIncludingPhased(p.ID())
			}
			p.RuntimeAbilities = slices.Clone(p.RuntimeAbilities)
			owned = true
		}
		switch rule := a.(type) {
		case *GenericTriggered:
			fresh := *rule
			fresh.source, fresh.controller = p.ID(), p.ControllerID()
			p.RuntimeAbilities[i] = &fresh
		case *ManaBonusAbility:
			fresh := *rule
			fresh.source, fresh.controller = p.ID(), p.ControllerID()
			p.RuntimeAbilities[i] = &fresh
		}
	}
	return p
}
