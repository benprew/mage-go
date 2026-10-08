package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func damageSourceFromExiledCard(object ExiledCard) DamageSource {
	source := DamageSource{id: object.Card.ID(), exileVisit: object.Visit, owner: object.Card.Owner()}
	if object.FaceDown {
		return source
	}
	source.card = object.Card.Copy()
	source.card.SetID(object.Card.ID())
	source.name = object.Card.Name()
	source.colors = slices.Clone(cardColors(object.Card))
	return source
}

// CaptureExiledDamageSource reads a live exile visit without private information.
// A card ID cannot select one of its separate visits.
func (g *Game) CaptureExiledDamageSource(visit uuid.UUID) DamageSource {
	if visit == uuid.Nil {
		return DamageSource{}
	}
	for _, object := range g.zones.exile {
		if object.Visit == visit {
			return damageSourceFromExiledCard(object)
		}
	}
	return DamageSource{}
}

func (zs *ZoneSystem) saveExileDeparture(object ExiledCard) {
	if zs.departedExileSources == nil {
		zs.departedExileSources = make(map[uuid.UUID]DamageSource)
	}
	zs.departedExileSources[object.Visit] = damageSourceFromExiledCard(object)
}

func (g *Game) exileReferenceInformation(saved DamageSource) DamageSource {
	if source := g.CaptureExiledDamageSource(saved.ExileVisit()); source.SameObject(saved) {
		return source
	}
	if source, ok := g.zones.departedExileSources[saved.ExileVisit()]; ok && source.SameObject(saved) {
		return cloneDamageSource(source)
	}
	return cloneDamageSource(saved)
}

type stackCardBindingReferences interface{ stackCardBindings() []capturedCard }

func (e *eventCardCapture) stackCardBindings() []capturedCard {
	if e.binding == nil {
		return nil
	}
	return []capturedCard{e.value}
}

func (e *sequenceEffect) stackCardBindings() []capturedCard {
	var identities []capturedCard
	for _, effect := range e.effects {
		if references, ok := effect.(stackCardBindingReferences); ok {
			identities = append(identities, references.stackCardBindings()...)
		}
	}
	return identities
}

// StackBoundExileSources reads exact exile targets and saved card bindings.
// Other source categories need separate enumeration services.
func (g *Game) StackBoundExileSources() []DamageSource {
	var sources []DamageSource
	for _, object := range g.stack.handReferenceObjects() {
		var identities []capturedCard
		for _, id := range object.Targets {
			if identity, ok := object.targetExileCards[id]; ok && identity.id == id && identity.zone == core.ZoneExile {
				identities = append(identities, identity)
			}
		}
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
			if references, ok := effect.(stackCardBindingReferences); ok {
				identities = append(identities, references.stackCardBindings()...)
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
			if identity.zone != core.ZoneExile || identity.id == uuid.Nil || identity.visit == uuid.Nil {
				continue
			}
			source := g.CaptureExiledDamageSource(identity.visit)
			if !source.Valid() {
				source = cloneDamageSource(g.zones.departedExileSources[identity.visit])
			}
			if source.ID() == identity.id && source.hasExactIdentity() && !slices.ContainsFunc(sources, source.SameObject) {
				sources = append(sources, source)
			}
		}
	}
	return sources
}

// ChooseDamageSourceFromStackExileBindings uses visits saved by stack objects.
// An exile source does not become a later spell or permanent source.
func ChooseDamageSourceFromStackExileBindings(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, stackExileBindings: true}
}
