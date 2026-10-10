package mage

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithLayeredAbilityTextRevision permits layer 3 changes to a printed static rule.
// The donor rule changes before its ability grants apply in layer 6.
func (a *StaticAbilityHolder) WithLayeredAbilityTextRevision() *StaticAbilityHolder {
	a.WithAbilityTextRevision()
	a.abilityTextLayer = true
	return a
}

type staticTextContract uint8

const (
	staticAbilityTextContract staticTextContract = iota
	staticGrantTextContract
	staticPTTextContract
	staticLandTypeTextContract
)

type printedStaticAbilityTextEffect struct {
	effectSource
	subject   capturedPermanent
	abilityID uuid.UUID
	duration  core.Duration
	changes   []TextChange
	contract  staticTextContract
}

func layeredStaticAbilityEligible(a *StaticAbilityHolder) bool {
	return a.abilityTextLayer && a.abilityTextRevision && a.supportsAbilityTextRevision()
}

// AddPrintedStaticAbilityTextEffect changes one printed donor rule in layer 3.
// Each pass starts with a saved baseline so expiry removes earlier changes.
func (g *Game) AddPrintedStaticAbilityTextEffect(source *Permanent, abilityID uuid.UUID, duration core.Duration, changes ...TextChange) error {
	return g.addPrintedStaticTextEffect(source, abilityID, duration, staticAbilityTextContract, changes...)
}

func (e *printedStaticAbilityTextEffect) eligible(a *StaticAbilityHolder) bool {
	switch e.contract {
	case staticGrantTextContract:
		return a.grantTextLayer && a.grantTextRevision && a.supportsGrantTextRevision()
	case staticPTTextContract:
		return a.ptTextLayer && a.ptTextRevision && a.supportsLayeredPTTextRevision()
	case staticLandTypeTextContract:
		return a.landTypeTextLayer && a.landTypeTextRevision && a.supportsLandTypeTextRevision()
	}
	return layeredStaticAbilityEligible(a)
}

func (g *Game) addPrintedStaticTextEffect(source *Permanent, abilityID uuid.UUID, duration core.Duration, kind staticTextContract, changes ...TextChange) error {
	contract := &printedStaticAbilityTextEffect{contract: kind}
	if source == nil || (duration != core.Indefinite && duration != core.EndOfTurn) || len(changes) == 0 {
		return fmt.Errorf("static ability text layer requires a source, changes and indefinite or end-of-turn duration")
	}
	current := g.FindPermanent(source.ID())
	if current == nil || current.incarnationID != source.incarnationID || current.FaceDown {
		return fmt.Errorf("static ability text layer requires the original face-up source")
	}
	for _, change := range changes {
		if change.kind != 1 && change.kind != 2 {
			return fmt.Errorf("static ability text layer requires validated color or basic land words")
		}
	}
	probe := g.Clone()
	if err := probe.revisePrintedStaticAbilityWithRefresh(probe.FindPermanent(source.ID()), abilityID, contract.eligible, false, changes...); err != nil {
		return err
	}
	effect := &printedStaticAbilityTextEffect{
		subject:   capturedPermanent{id: current.ID(), incarnation: current.incarnationID},
		abilityID: abilityID, duration: duration, changes: slices.Clone(changes), contract: kind,
	}
	effect.SetSourceID(current.ID())
	g.AddContinuousEffect(effect)
	return nil
}

func (*printedStaticAbilityTextEffect) Text() string                 { return "change printed static ability text" }
func (*printedStaticAbilityTextEffect) Properties() EffectProperties { return EffectProperties{} }
func (*printedStaticAbilityTextEffect) GetLayer() core.Layer         { return core.LayerText }
func (e *printedStaticAbilityTextEffect) GetDuration() core.Duration { return e.duration }
func (e *printedStaticAbilityTextEffect) IsActive(g *Game) bool {
	p := g.FindPermanent(e.subject.id)
	return p != nil && p.incarnationID == e.subject.incarnation && !p.FaceDown && slices.ContainsFunc(p.RuntimeAbilities, func(a Ability) bool {
		rule, ok := a.(*StaticAbilityHolder)
		return ok && rule.AbilityID() == e.abilityID && e.eligible(rule)
	})
}
func (e *printedStaticAbilityTextEffect) ApplyContinuous(ctx *EffectContext) error {
	g := ctx.Game
	if !e.IsActive(g) {
		return nil
	}
	p := g.MutablePermanent(e.subject.id)
	if p.layeredStaticAbilityOrigins[e.abilityID] == nil {
		p.layeredStaticAbilityOrigins = maps.Clone(p.layeredStaticAbilityOrigins)
		if p.layeredStaticAbilityOrigins == nil {
			p.layeredStaticAbilityOrigins = make(map[uuid.UUID]*StaticAbilityHolder)
		}
		for _, a := range p.RuntimeAbilities {
			if rule, ok := a.(*StaticAbilityHolder); ok && rule.AbilityID() == e.abilityID {
				p.layeredStaticAbilityOrigins[e.abilityID] = rule
			}
		}
	}
	return g.revisePrintedStaticAbilityWithRefresh(p, e.abilityID, e.eligible, false, e.changes...)
}
func (e *printedStaticAbilityTextEffect) cloneEffect() ContinuousEffect {
	clone := *e
	clone.changes = slices.Clone(e.changes)
	return &clone
}

func (g *Game) resetPrintedStaticAbilityTextLayers() {
	for _, p := range g.zones.battlefield {
		if p.PhasedOut || len(p.layeredStaticAbilityOrigins) == 0 {
			continue
		}
		p = g.MutablePermanent(p.ID())
		p.RuntimeAbilities = slices.Clone(p.RuntimeAbilities)
		registrations := slices.Clone(g.effects.effects)
		for id, origin := range p.layeredStaticAbilityOrigins {
			restored := *origin
			restored.source = p.ID()
			restored.controller = p.ControllerID()
			for i, a := range p.RuntimeAbilities {
				if a.AbilityID() == id {
					p.RuntimeAbilities[i] = &restored
				}
			}
			part := 0
			for i, effect := range registrations {
				if !g.printedStaticRegistration(effect, p, id) {
					continue
				}
				fresh := g.bindCharacteristicEffect(restored.Effects[part], p)
				g.setEffectSource(fresh, p.ID())
				registrations[i] = printedStaticEffect(fresh, p, &restored)
				part++
			}
		}
		g.effects.effects = registrations
		p.layeredStaticAbilityOrigins = nil
	}
}

// A changed controller needs a separate holder because game clones share it.
func (g *Game) isolatePrintedStaticAbilityContext(p *Permanent) *Permanent {
	owned := false
	for i, ability := range p.RuntimeAbilities {
		rule, ok := ability.(*StaticAbilityHolder)
		if !ok || (!rule.abilityTextLayer && !rule.grantTextLayer && !rule.ptTextLayer && !rule.landTypeTextLayer) || (rule.Source() == p.ID() && rule.Controller() == p.ControllerID()) {
			continue
		}
		if !owned {
			if current := g.FindPermanentIncludingPhased(p.ID()); current != nil && current.incarnationID == p.incarnationID {
				p = g.mutablePermanentIncludingPhased(p.ID())
			}
			p.RuntimeAbilities = slices.Clone(p.RuntimeAbilities)
			g.effects.effects = slices.Clone(g.effects.effects)
			owned = true
		}
		fresh := *rule
		fresh.source = p.ID()
		fresh.controller = p.ControllerID()
		p.RuntimeAbilities[i] = &fresh
		for j, effect := range g.effects.effects {
			if g.printedStaticRegistration(effect, p, rule.AbilityID()) {
				registration := *effect.(*printedAbilityContinuousEffect)
				registration.declaration = &fresh
				g.effects.effects[j] = &registration
			}
		}
	}
	return p
}
