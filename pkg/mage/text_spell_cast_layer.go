package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithLayeredSpellCastTextRevision permits layer 3 changes to a printed spell-cast event.
// Pending triggers keep their text and event facts from detection.
func (t *GenericTriggered) WithLayeredSpellCastTextRevision() *GenericTriggered {
	t.WithSpellCastTextRevision()
	t.spellCastTextLayer = true
	return t
}

func layeredSpellCastEligible(a Ability) bool {
	rule, ok := a.(*GenericTriggered)
	return ok && rule.spellCastTextLayer && rule.spellCastTextRevision && rule.supportsSpellCastTextRevision()
}

type printedSpellCastTextEffect struct {
	effectSource
	subject   capturedPermanent
	abilityID uuid.UUID
	duration  core.Duration
	changes   []TextChange
}

// AddPrintedSpellCastTextEffect changes one printed spell-cast declaration in layer 3.
// Each pass uses its saved baseline so expiry removes earlier changes.
func (g *Game) AddPrintedSpellCastTextEffect(source *Permanent, abilityID uuid.UUID, duration core.Duration, changes ...TextChange) error {
	if source == nil || (duration != core.Indefinite && duration != core.EndOfTurn) || len(changes) == 0 {
		return fmt.Errorf("spell-cast text layer requires a source, changes and indefinite or end-of-turn duration")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("spell-cast text layer requires the original face-up source")
	}
	if !slices.ContainsFunc(current.RuntimeAbilities, func(a Ability) bool { return a.AbilityID() == abilityID && layeredSpellCastEligible(a) }) {
		return fmt.Errorf("spell-cast text layer requires an opted-in printed declaration")
	}
	for _, change := range changes {
		if change.kind != 1 && change.kind != 2 {
			return fmt.Errorf("spell-cast text layer requires validated color or basic land words")
		}
	}
	probe := g.Clone()
	if err := probe.revisePrintedSpellCastLayer(probe.FindPermanent(source.ID()), abilityID, changes...); err != nil {
		return err
	}
	effect := &printedSpellCastTextEffect{subject: capturedPermanent{id: current.ID(), incarnation: current.incarnationID}, abilityID: abilityID, duration: duration, changes: slices.Clone(changes)}
	effect.SetSourceID(current.ID())
	if err := g.prepareAddedResolvingCopyText(current, effect); err != nil {
		return err
	}
	g.AddContinuousEffect(effect)
	return nil
}

func (*printedSpellCastTextEffect) Text() string                 { return "change printed spell-cast text" }
func (*printedSpellCastTextEffect) Properties() EffectProperties { return EffectProperties{} }
func (*printedSpellCastTextEffect) GetLayer() core.Layer         { return core.LayerText }
func (e *printedSpellCastTextEffect) GetDuration() core.Duration { return e.duration }
func (e *printedSpellCastTextEffect) IsActive(g *Game) bool {
	p := g.FindPermanent(e.subject.id)
	return p != nil && p.incarnationID == e.subject.incarnation && !p.FaceDown && slices.ContainsFunc(p.RuntimeAbilities, func(a Ability) bool { return a.AbilityID() == e.abilityID && layeredSpellCastEligible(a) })
}
func (e *printedSpellCastTextEffect) ApplyContinuous(ctx *EffectContext) error {
	g := ctx.Game
	if !e.IsActive(g) {
		return nil
	}
	p := g.MutablePermanent(e.subject.id)
	if _, exists := p.layeredSpellCastOrigins[e.abilityID]; !exists {
		p.layeredSpellCastOrigins = maps.Clone(p.layeredSpellCastOrigins)
		if p.layeredSpellCastOrigins == nil {
			p.layeredSpellCastOrigins = make(map[uuid.UUID]*GenericTriggered)
		}
		for _, a := range p.RuntimeAbilities {
			if a.AbilityID() != e.abilityID {
				continue
			}
			p.layeredSpellCastOrigins[e.abilityID] = a.(*GenericTriggered)
		}
	}
	return g.revisePrintedSpellCastLayer(p, e.abilityID, e.changes...)
}
func (e *printedSpellCastTextEffect) cloneEffect() ContinuousEffect {
	clone := *e
	clone.changes = slices.Clone(e.changes)
	return &clone
}

func (g *Game) resetPrintedSpellCastTextLayers() {
	for _, p := range g.zones.battlefield {
		if p.PhasedOut || len(p.layeredSpellCastOrigins) == 0 {
			continue
		}
		p = g.MutablePermanent(p.ID())
		p.RuntimeAbilities = slices.Clone(p.RuntimeAbilities)
		p.revisedPrintedTriggers = maps.Clone(p.revisedPrintedTriggers)
		for id, origin := range p.layeredSpellCastOrigins {
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
		p.layeredSpellCastOrigins = nil
	}
}

// A changed context needs a separate declaration because game clones share it.
func (g *Game) isolatePrintedSpellCastContext(p *Permanent) *Permanent {
	owned := false
	for i, a := range p.RuntimeAbilities {
		if !layeredSpellCastEligible(a) || (a.Source() == p.ID() && a.Controller() == p.ControllerID()) {
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

func (g *Game) revisePrintedSpellCastLayer(source *Permanent, id uuid.UUID, changes ...TextChange) error {
	var original spellCastPattern
	if source != nil {
		if current := g.FindPermanent(source.ID()); current != nil {
			for _, a := range current.RuntimeAbilities {
				if a.AbilityID() == id && layeredSpellCastEligible(a) {
					original = a.(*GenericTriggered).eventDefinition.(spellCastPattern)
				}
			}
		}
	}
	return g.revisePrintedTrigger(source, id, func(t *GenericTriggered) bool {
		if !layeredSpellCastEligible(t) {
			return false
		}
		pattern := t.eventDefinition.(spellCastPattern)
		return sameSpellCastTextContext(original, pattern)
	}, changes...)
}

func sameSpellCastTextContext(a, b spellCastPattern) bool {
	a.contextualFilter, b.contextualFilter = nil, nil
	a.color, b.color = core.Colorless, core.Colorless
	a.spell, b.spell = nil, nil
	a.caster, b.caster = nil, nil
	a.manaSpent, b.manaSpent = nil, nil
	a.colorsSpent, b.colorsSpent = nil, nil
	a.announcedX, b.announcedX = nil, nil
	return a == b
}
