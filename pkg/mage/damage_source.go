package mage

import (
	"maps"
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func cloneDamageSource(s DamageSource) DamageSource {
	s.permanentTargets = slices.Clone(s.permanentTargets)
	s.colors = slices.Clone(s.colors)
	s.keywords = maps.Clone(s.keywords)
	if s.card != nil {
		s.card = s.card.Copy()
		s.card.SetID(s.id)
	}
	return s
}

// DamageSource records the source state used by one damage instruction.
type DamageSource struct {
	permanentTargets                   []capturedPermanent
	id, incarnation, controller, owner uuid.UUID
	stackID                            uuid.UUID
	commandVisit                       uuid.UUID
	exileVisit                         uuid.UUID
	graveyardVisit                     uuid.UUID
	handVisit                          uuid.UUID
	name                               string
	colors                             []Color
	keywords                           map[Keyword]bool
	card                               Card
}

func (s DamageSource) Valid() bool            { return s.id != uuid.Nil }
func (s DamageSource) ID() uuid.UUID          { return s.id }
func (s DamageSource) Incarnation() uuid.UUID { return s.incarnation }

// StackID keeps separate casts of the same card apart.
func (s DamageSource) StackID() uuid.UUID { return s.stackID }

// TargetsPermanent checks the saved target visit of a spell source.
// Target legality does not remove the announced reference.
func (s DamageSource) TargetsPermanent(permanent *Permanent) bool {
	if permanent == nil || s.stackID == uuid.Nil || !s.hasExactIdentity() {
		return false
	}
	return slices.Contains(s.permanentTargets, capturedPermanent{id: permanent.ID(), incarnation: permanent.incarnationID})
}

// CommandVisit keeps separate command-zone entries of one object apart.
func (s DamageSource) CommandVisit() uuid.UUID { return s.commandVisit }

// ExileVisit keeps separate exile entries of one card apart.
func (s DamageSource) ExileVisit() uuid.UUID { return s.exileVisit }

// GraveyardVisit keeps separate graveyard entries of one card apart.
func (s DamageSource) GraveyardVisit() uuid.UUID { return s.graveyardVisit }

// HandVisit keeps separate hand entries of one card apart.
func (s DamageSource) HandVisit() uuid.UUID { return s.handVisit }

func (s DamageSource) hasExactIdentity() bool {
	count := 0
	for _, id := range []uuid.UUID{s.incarnation, s.stackID, s.commandVisit, s.exileVisit, s.graveyardVisit, s.handVisit} {
		if id != uuid.Nil {
			count++
		}
	}
	return s.Valid() && count == 1
}

// SameObject compares exact source identities. Missing identities cannot match.
func (s DamageSource) SameObject(other DamageSource) bool {
	return s.hasExactIdentity() && other.hasExactIdentity() && s.id == other.id &&
		s.incarnation == other.incarnation && s.stackID == other.stackID && s.commandVisit == other.commandVisit && s.exileVisit == other.exileVisit && s.graveyardVisit == other.graveyardVisit && s.handVisit == other.handVisit
}
func (s DamageSource) Controller() uuid.UUID     { return s.controller }
func (s DamageSource) Owner() uuid.UUID          { return s.owner }
func (s DamageSource) Name() string              { return s.name }
func (s DamageSource) HasKeyword(k Keyword) bool { return s.keywords[k] }
func (s DamageSource) HasColor(c Color) bool     { return slices.Contains(s.colors, c) }
func (s DamageSource) HasType(t CardType) bool   { return s.card != nil && s.card.HasType(t) }
func (s DamageSource) HasSubType(st string) bool { return s.card != nil && s.card.HasSubType(st) }

func damageSourceFromPermanent(p *Permanent) DamageSource {
	s := DamageSource{id: p.ID(), incarnation: p.incarnationID, controller: p.ControllerID(), owner: p.Card.Owner(), name: p.Name(), card: damageSourceCardFromPermanent(p), colors: slices.Clone(p.Colors()), keywords: make(map[Keyword]bool)}
	if p.FaceDown {
		s.name = ""
	}
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
	if p.FaceDown {
		s.name = ""
	}
	for k := range Keyword(NumAttrs) {
		if l.ViewHasKeyword(k) {
			s.keywords[k] = true
		}
	}
	return s
}

func damageSourceCardFromPermanent(p *Permanent) Card {
	if p.FaceDown {
		// Choices must not expose the card's hidden characteristics.
		card := NewCreature("", "", 2, 2)
		card.SetID(p.ID())
		card.SetOwner(p.Card.Owner())
		return card
	}
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
		colors := cardColors(card)
		if g.resolution.ColorSourceID() == id && g.resolution.ColorOverride() != nil {
			colors = *g.resolution.ColorOverride()
		}
		return DamageSource{id: id, stackID: g.resolution.resolvingStackID, controller: controller, owner: card.Owner(), name: card.Name(), card: copied, colors: slices.Clone(colors), permanentTargets: slices.Clone(g.resolution.resolvingPermanentTargets)}
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

// damageFilterSource keeps a returned card separate from the original source.
func (g *Game) damageFilterSource(id uuid.UUID, source DamageSource) *Permanent {
	if source.Incarnation() == uuid.Nil {
		if source.Valid() {
			return nil
		}
		return g.FindPermanent(id)
	}
	if source.ID() != id {
		return nil
	}
	if p := g.FindPermanentIncludingPhased(id); p != nil && p.incarnationID == source.Incarnation() {
		if !p.PhasedOut {
			return p
		}
		snapshot := &Permanent{}
		clonePermanentInto(snapshot, p)
		return snapshot
	}
	if lki := g.zones.LKIByIncarnation(source.Incarnation()); lki != nil && lki.ID == id && lki.Snapshot != nil {
		snapshot := &Permanent{}
		clonePermanentInto(snapshot, lki.Snapshot)
		return snapshot
	}
	return nil
}
