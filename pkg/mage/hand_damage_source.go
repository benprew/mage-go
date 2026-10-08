package mage

import (
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func damageSourceFromHandCard(card Card, visit uuid.UUID, public bool) DamageSource {
	if !public {
		return DamageSource{id: card.ID(), handVisit: visit, owner: card.Owner()}
	}
	snapshot := card.Copy()
	snapshot.SetID(card.ID())
	return DamageSource{id: card.ID(), handVisit: visit, owner: card.Owner(), name: card.Name(), colors: slices.Clone(cardColors(card)), card: snapshot}
}

// CaptureHandDamageSource reads one live hand visit.
// A physical card ID cannot select a visit.
func (g *Game) CaptureHandDamageSource(visit uuid.UUID) DamageSource {
	if visit == uuid.Nil {
		return DamageSource{}
	}
	for _, player := range g.players {
		visitor, ok := player.(interface{ handVisit(uuid.UUID) uuid.UUID })
		if !ok {
			continue
		}
		for _, card := range player.Hand() {
			if visitor.handVisit(card.ID()) == visit {
				return damageSourceFromHandCard(card, visit, g.IsHandRevealed(player.PlayerID()) || g.stack.handVisitRevealed(card.ID(), visit))
			}
		}
	}
	return DamageSource{}
}

func (p *BasePlayer) saveHandDeparture(card Card) {
	visit := p.handVisit(card.ID())
	if visit == uuid.Nil {
		return
	}
	p.departedHandSources = maps.Clone(p.departedHandSources)
	if p.departedHandSources == nil {
		p.departedHandSources = make(map[uuid.UUID]DamageSource)
	}
	p.departedHandSources[visit] = damageSourceFromHandCard(card, visit, p.handPublic || p.handStack.handVisitRevealed(card.ID(), visit))
}

func (p *BasePlayer) handDepartures() map[uuid.UUID]DamageSource {
	return p.departedHandSources
}

func (g *Game) handVisitInformation(id, visit uuid.UUID) DamageSource {
	if id == uuid.Nil || visit == uuid.Nil {
		return DamageSource{}
	}
	if source := g.CaptureHandDamageSource(visit); source.ID() == id {
		return source
	}
	for _, player := range g.players {
		if history, ok := player.(interface {
			handDepartures() map[uuid.UUID]DamageSource
		}); ok {
			if source, found := history.handDepartures()[visit]; found && source.ID() == id {
				return cloneDamageSource(source)
			}
		}
	}
	return DamageSource{}
}

func (g *Game) handReferenceInformation(saved DamageSource) DamageSource {
	if source := g.handVisitInformation(saved.ID(), saved.HandVisit()); source.SameObject(saved) {
		return source
	}
	return cloneDamageSource(saved)
}

// StackHandSources reads exact hand visits in saved card bindings.
// Departure information alone cannot make a source available for a choice.
func (g *Game) StackHandSources() []DamageSource {
	var sources []DamageSource
	appendSource := func(source DamageSource) {
		if source.hasExactIdentity() && !slices.ContainsFunc(sources, source.SameObject) {
			sources = append(sources, source)
		}
	}
	for _, object := range g.stack.handReferenceObjects() {
		identities := slices.Clone(object.revealedHandCards)
		for binding, identity := range object.capturedCards {
			if binding != nil {
				identities = append(identities, identity)
			}
		}
		for binding, values := range object.capturedCardSets {
			if binding != nil {
				identities = append(identities, values...)
			}
		}
		for _, effect := range object.Effects {
			if refs, ok := effect.(stackCardBindingReferences); ok {
				identities = append(identities, refs.stackCardBindings()...)
			}
		}
		// Map order must not change the choices shown to the player.
		slices.SortFunc(identities, func(a, b capturedCard) int {
			if order := slices.Compare(a.id[:], b.id[:]); order != 0 {
				return order
			}
			return slices.Compare(a.visit[:], b.visit[:])
		})
		for _, identity := range identities {
			if identity.zone == core.ZoneHand {
				appendSource(g.handVisitInformation(identity.id, identity.visit))
			}
		}
	}
	return sources
}

// ChooseDamageSourceFromStackHandReferences uses visits saved on the stack.
// Other source categories need separate enumeration services.
func ChooseDamageSourceFromStackHandReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, stackHandReferences: true}
}

func (p *BasePlayer) setHandPublic(public bool) { p.handPublic = public }

func (g *Game) syncHandVisibility() {
	for _, player := range g.players {
		if state, ok := player.(interface{ setHandPublic(bool) }); ok {
			state.setHandPublic(g.IsHandRevealed(player.PlayerID()))
		}
	}
}

func (g *Game) handVisit(id uuid.UUID) uuid.UUID {
	for _, player := range g.players {
		if visitor, ok := player.(interface{ handVisit(uuid.UUID) uuid.UUID }); ok {
			for _, card := range player.Hand() {
				if card.ID() == id {
					return visitor.handVisit(id)
				}
			}
		}
	}
	return uuid.Nil
}

func (p *BasePlayer) handIsPublic() bool { return p.handPublic }

func (s *Stack) handVisitRevealed(id, visit uuid.UUID) bool {
	if s == nil || visit == uuid.Nil {
		return false
	}
	for _, object := range s.handReferenceObjects() {
		if !object.ownsHandReveals {
			continue
		}
		for _, identity := range object.revealedHandCards {
			if identity.id == id && identity.visit == visit && identity.zone == core.ZoneHand {
				return true
			}
		}
	}
	return false
}

func (p *BasePlayer) setHandStack(stack *Stack) { p.handStack = stack }

func (g *Game) bindHandStack() {
	for _, player := range g.players {
		if state, ok := player.(interface{ setHandStack(*Stack) }); ok {
			state.setHandStack(g.stack)
		}
	}
}

func (s *Stack) handReferenceObjects() []*StackObject {
	if len(s.resolvingHandObjects) == 0 {
		return s.objects
	}
	return slices.Concat(s.objects, s.resolvingHandObjects)
}
