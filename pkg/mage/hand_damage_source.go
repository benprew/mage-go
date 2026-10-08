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
	return g.CaptureHandDamageSourceFor(uuid.Nil, visit)
}

// CaptureHandDamageSourceFor includes the hand player's inspection information.
// Other viewers receive only public facts. This query does not reveal a card.
func (g *Game) CaptureHandDamageSourceFor(viewer, visit uuid.UUID) DamageSource {
	return g.captureHandDamageSource(viewer, visit, false)
}

func (g *Game) captureHandDamageSource(viewer, visit uuid.UUID, rules bool) DamageSource {
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
				return damageSourceFromHandCard(card, visit, rules || viewer == player.PlayerID() || g.IsHandRevealed(player.PlayerID()) || g.stack.handVisitRevealed(card.ID(), visit))
			}
		}
	}
	return DamageSource{}
}

type handDeparture struct {
	source DamageSource
	public bool
}

func (p *BasePlayer) saveHandDeparture(card Card) {
	visit := p.handVisit(card.ID())
	if visit == uuid.Nil {
		return
	}
	p.departedHandSources = maps.Clone(p.departedHandSources)
	if p.departedHandSources == nil {
		p.departedHandSources = make(map[uuid.UUID]handDeparture)
	}
	p.departedHandSources[visit] = handDeparture{
		source: damageSourceFromHandCard(card, visit, true),
		public: p.handPublic || p.handStack.handVisitRevealed(card.ID(), visit),
	}
}

func (p *BasePlayer) handDepartures() map[uuid.UUID]handDeparture {
	return p.departedHandSources
}

func (g *Game) handVisitInformation(id, visit uuid.UUID) DamageSource {
	return g.handVisitInformationFor(uuid.Nil, id, visit, false)
}

func (g *Game) handVisitInformationFor(viewer, id, visit uuid.UUID, rules bool) DamageSource {
	if id == uuid.Nil || visit == uuid.Nil {
		return DamageSource{}
	}
	if source := g.captureHandDamageSource(viewer, visit, rules); source.ID() == id {
		return source
	}
	for _, player := range g.players {
		if history, ok := player.(interface {
			handDepartures() map[uuid.UUID]handDeparture
		}); ok {
			if departure, found := history.handDepartures()[visit]; found && departure.source.ID() == id {
				if rules || viewer == player.PlayerID() || departure.public {
					return cloneDamageSource(departure.source)
				}
				return hiddenHandSource(departure.source)
			}
		}
	}
	return DamageSource{}
}

func (g *Game) handReferenceInformation(saved DamageSource) DamageSource {
	if source := g.handVisitInformation(saved.ID(), saved.HandVisit()); source.SameObject(saved) {
		return source
	}
	return hiddenHandSource(saved)
}

func (g *Game) handReferenceInformationFor(viewer uuid.UUID, saved DamageSource) DamageSource {
	if viewer == uuid.Nil {
		return g.handReferenceInformation(saved)
	}
	if source := g.handVisitInformationFor(viewer, saved.ID(), saved.HandVisit(), false); source.SameObject(saved) {
		return source
	}
	return hiddenHandSource(saved)
}

func hiddenHandSource(source DamageSource) DamageSource {
	return DamageSource{id: source.ID(), handVisit: source.HandVisit(), owner: source.Owner()}
}

// StackHandSources reads exact hand visits in saved card bindings.
// Departure information alone cannot make a source available for a choice.
func (g *Game) StackHandSources() []DamageSource {
	return g.StackHandSourcesFor(uuid.Nil)
}

// StackHandSourcesFor reads references with the viewer's hand inspection rights.
// An unreferenced card cannot become a choice through inspection alone.
func (g *Game) StackHandSourcesFor(viewer uuid.UUID) []DamageSource {
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
				appendSource(g.handVisitInformationFor(viewer, identity.id, identity.visit, false))
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

// ChooseDamageSourceFromInspectableStackHandReferences uses the choosing player's information.
// Public queries and other source categories keep their separate contracts.
func ChooseDamageSourceFromInspectableStackHandReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, stackHandReferences: true, inspectHandSources: true}
}

func (g *Game) damageSourceRuleInformation(saved DamageSource) DamageSource {
	if saved.hasExactIdentity() && saved.HandVisit() != uuid.Nil {
		if source := g.handVisitInformationFor(uuid.Nil, saved.ID(), saved.HandVisit(), true); source.SameObject(saved) {
			return source
		}
	}
	return g.damageReferenceInformation(saved)
}
