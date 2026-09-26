package mage

import (
	"maps"
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func cloneDamageSource(s DamageSource) DamageSource {
	s.colors = slices.Clone(s.colors)
	s.keywords = maps.Clone(s.keywords)
	if s.card != nil {
		s.card = s.card.Copy()
	}
	return s
}

// DamageSource records the source state used by one damage instruction.
type DamageSource struct {
	id, incarnation, controller, owner uuid.UUID
	name                               string
	colors                             []Color
	keywords                           map[Keyword]bool
	card                               Card
}

func (s DamageSource) Valid() bool               { return s.id != uuid.Nil }
func (s DamageSource) ID() uuid.UUID             { return s.id }
func (s DamageSource) Incarnation() uuid.UUID    { return s.incarnation }
func (s DamageSource) Controller() uuid.UUID     { return s.controller }
func (s DamageSource) Owner() uuid.UUID          { return s.owner }
func (s DamageSource) Name() string              { return s.name }
func (s DamageSource) HasKeyword(k Keyword) bool { return s.keywords[k] }
func (s DamageSource) HasColor(c Color) bool     { return slices.Contains(s.colors, c) }
func (s DamageSource) HasType(t CardType) bool   { return s.card != nil && s.card.HasType(t) }

func damageSourceFromPermanent(p *Permanent) DamageSource {
	s := DamageSource{id: p.ID(), incarnation: p.incarnationID, controller: p.ControllerID(), owner: p.Card.Owner(), name: p.Name(), card: damageSourceCardFromPermanent(p), colors: slices.Clone(p.Colors()), keywords: make(map[Keyword]bool)}
	for k := range Keyword(NumAttrs) {
		if p.HasKeyword(k) {
			s.keywords[k] = true
		}
	}
	return s
}

func damageSourceFromLKI(l *PermanentLKI) DamageSource {
	if l == nil || l.Snapshot == nil {
		return DamageSource{}
	}
	p := l.Snapshot
	s := DamageSource{id: l.ID, incarnation: p.incarnationID, controller: p.ControllerID(), owner: l.Owner, name: l.Name, card: damageSourceCardFromPermanent(p), colors: slices.Clone(p.Colors()), keywords: make(map[Keyword]bool)}
	for k := range Keyword(NumAttrs) {
		if l.ViewHasKeyword(k) {
			s.keywords[k] = true
		}
	}
	return s
}

func damageSourceCardFromPermanent(p *Permanent) Card {
	card := p.Card.Copy()
	card.SetID(p.ID())
	if base, ok := card.(*BaseCard); ok {
		base.types = nil
		for cardType := TypeCreature; cardType <= TypePlaneswalker; cardType++ {
			if p.HasType(cardType) {
				base.types = append(base.types, cardType)
			}
		}
		base.subTypes = slices.Clone(p.SubTypes())
	}
	return card
}

// CaptureDamageSource freezes the current source or the resolving ability's source incarnation.
func (g *Game) CaptureDamageSource(id uuid.UUID) DamageSource {
	if id == uuid.Nil {
		return DamageSource{}
	}
	if g.resolution.sourceCaptured && g.resolution.sourceIncarnation != uuid.Nil {
		inc := g.resolution.sourceIncarnation
		if p := g.FindPermanent(id); p != nil && p.incarnationID == inc {
			return damageSourceFromPermanent(p)
		}
		if lki := g.zones.LKIByIncarnation(inc); lki != nil && lki.ID == id {
			return damageSourceFromLKI(lki)
		}
	}
	if p := g.FindPermanent(id); p != nil {
		return damageSourceFromPermanent(p)
	}
	if card := g.resolution.ResolvingCard(); card != nil && card.ID() == id {
		controller := g.resolution.resolvingController
		if controller == uuid.Nil {
			controller = card.Owner()
		}
		copied := card.Copy()
		copied.SetID(id)
		return DamageSource{id: id, controller: controller, owner: card.Owner(), name: card.Name(), card: copied, colors: slices.Clone(g.EffectiveColors(id))}
	}
	return DamageSource{}
}

func protectedFromDamageSource(perm *Permanent, source DamageSource) bool {
	if perm.FaceDown || !source.Valid() {
		return false
	}
	for _, ability := range perm.RuntimeAbilities {
		protection, ok := UnwrapAbility(ability).(*ProtectionAbility)
		if !ok || (protection.ExemptCardID != uuid.Nil && protection.ExemptCardID == source.ID()) {
			continue
		}
		if len(protection.FromColors) > 0 {
			if slices.ContainsFunc(protection.FromColors, source.HasColor) {
				return true
			}
		} else if source.card != nil && protection.Blocks(source.card) {
			return true
		}
	}
	return false
}
