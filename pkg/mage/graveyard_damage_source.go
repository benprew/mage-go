package mage

import (
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func damageSourceFromGraveyardCard(card Card, visit uuid.UUID) DamageSource {
	snapshot := card.Copy()
	snapshot.SetID(card.ID())
	return DamageSource{id: card.ID(), graveyardVisit: visit, owner: card.Owner(), name: card.Name(), colors: slices.Clone(cardColors(card)), card: snapshot}
}

// CaptureGraveyardDamageSource reads one live graveyard visit.
// A physical card ID cannot select a visit.
func (g *Game) CaptureGraveyardDamageSource(visit uuid.UUID) DamageSource {
	if visit == uuid.Nil {
		return DamageSource{}
	}
	for _, player := range g.players {
		visitor, ok := player.(interface{ graveyardVisit(uuid.UUID) uuid.UUID })
		if !ok {
			continue
		}
		for _, card := range player.Graveyard() {
			if visitor.graveyardVisit(card.ID()) == visit {
				return damageSourceFromGraveyardCard(card, visit)
			}
		}
	}
	return DamageSource{}
}

func (p *BasePlayer) saveGraveyardDeparture(card Card) {
	visit := p.graveyardVisit(card.ID())
	if visit == uuid.Nil {
		return
	}
	p.departedGraveyardSources = maps.Clone(p.departedGraveyardSources)
	if p.departedGraveyardSources == nil {
		p.departedGraveyardSources = make(map[uuid.UUID]DamageSource)
	}
	p.departedGraveyardSources[visit] = damageSourceFromGraveyardCard(card, visit)
}

func (p *BasePlayer) graveyardDepartures() map[uuid.UUID]DamageSource {
	return p.departedGraveyardSources
}

func cloneZoneCards(cards []Card) []Card {
	if len(cards) == 0 {
		return nil
	}
	result := make([]Card, len(cards))
	for i, card := range cards {
		result[i] = card.Copy()
		result[i].SetID(card.ID())
	}
	return result
}

func (g *Game) graveyardVisitInformation(id, visit uuid.UUID) DamageSource {
	if id == uuid.Nil || visit == uuid.Nil {
		return DamageSource{}
	}
	if source := g.CaptureGraveyardDamageSource(visit); source.ID() == id {
		return source
	}
	for _, player := range g.players {
		if history, ok := player.(interface {
			graveyardDepartures() map[uuid.UUID]DamageSource
		}); ok {
			if source, found := history.graveyardDepartures()[visit]; found && source.ID() == id {
				return cloneDamageSource(source)
			}
		}
	}
	return DamageSource{}
}

func (g *Game) graveyardReferenceInformation(saved DamageSource) DamageSource {
	if source := g.graveyardVisitInformation(saved.ID(), saved.GraveyardVisit()); source.SameObject(saved) {
		return source
	}
	return cloneDamageSource(saved)
}

func (g *Game) captureGraveyardTargetSources(object *StackObject) {
	for _, id := range object.Targets {
		if _, saved := object.targetExileCards[id]; saved {
			continue
		}
		if _, saved := object.targetGraveyardSources[id]; saved {
			continue
		}
		visit := g.graveyardVisit(id)
		if captured, ok := object.targetIncarnations[id]; ok && captured != visit {
			continue
		}
		source := g.CaptureGraveyardDamageSource(visit)
		if !source.hasExactIdentity() {
			continue
		}
		object.targetGraveyardSources = maps.Clone(object.targetGraveyardSources)
		if object.targetGraveyardSources == nil {
			object.targetGraveyardSources = make(map[uuid.UUID]DamageSource)
		}
		object.targetGraveyardSources[id] = source
	}
}

// StackGraveyardSources reads exact graveyard targets and saved card bindings.
// Resolving frames keep their references until the body completes.
// Departure information alone cannot make a source available for a choice.
func (g *Game) StackGraveyardSources() []DamageSource {
	var sources []DamageSource
	appendSource := func(source DamageSource) {
		if source.hasExactIdentity() && !slices.ContainsFunc(sources, source.SameObject) {
			sources = append(sources, source)
		}
	}
	for _, object := range g.stack.referenceObjects() {
		for _, id := range object.Targets {
			saved, ok := object.targetGraveyardSources[id]
			if ok && saved.ID() == id && saved.GraveyardVisit() != uuid.Nil {
				appendSource(g.damageReferenceInformation(saved))
			}
		}
		var identities []capturedCard
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
			if identity.zone == core.ZoneGraveyard {
				appendSource(g.graveyardVisitInformation(identity.id, identity.visit))
			}
		}
	}
	return sources
}

// ChooseDamageSourceFromStackGraveyardReferences uses visits saved on the stack.
// Other source categories need separate enumeration services.
func ChooseDamageSourceFromStackGraveyardReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, stackGraveyardReferences: true}
}
