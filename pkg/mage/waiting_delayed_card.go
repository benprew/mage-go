package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func (e *capturedDelayedBody) stackCardBindings() []capturedCard {
	var identities []capturedCard
	for binding, identity := range e.cards {
		if binding != nil {
			identities = append(identities, identity)
		}
	}
	for binding, values := range e.cardSets {
		if binding != nil {
			identities = append(identities, values...)
		}
	}
	slices.SortFunc(identities, func(a, b capturedCard) int {
		if order := slices.Compare(a.id[:], b.id[:]); order != 0 {
			return order
		}
		if order := slices.Compare(a.visit[:], b.visit[:]); order != 0 {
			return order
		}
		return int(a.zone) - int(b.zone)
	})
	return identities
}

func (g *Game) waitingDelayedCardSources(viewer uuid.UUID, zone core.Zone) []DamageSource {
	var sources []DamageSource
	appendEffects := func(effects []Effect) {
		for _, effect := range effects {
			refs, ok := effect.(stackCardBindingReferences)
			if !ok {
				continue
			}
			for _, identity := range refs.stackCardBindings() {
				if identity.zone != zone || identity.id == uuid.Nil || identity.visit == uuid.Nil {
					continue
				}
				var source DamageSource
				switch zone {
				case core.ZoneHand:
					source = g.handVisitInformationFor(viewer, identity.id, identity.visit, false)
				case core.ZoneGraveyard:
					source = g.graveyardVisitInformation(identity.id, identity.visit)
				default:
					source = g.CaptureExiledDamageSource(identity.visit)
					if !source.Valid() {
						source = cloneDamageSource(g.zones.departedExileSources[identity.visit])
					}
				}
				if source.ID() == identity.id && source.hasExactIdentity() && !slices.ContainsFunc(sources, source.SameObject) {
					sources = append(sources, source)
				}
			}
		}
	}
	for _, delayed := range g.triggers.Delayed() {
		if delayed == nil {
			continue
		}
		appendEffects(delayed.Effects)
		for _, mode := range delayed.modes {
			appendEffects(mode.Effects)
		}
	}
	return sources
}

// WaitingDelayedHandSourcesFor reads saved hand visits with the viewer's inspection rights.
// A saved reference cannot reveal a hidden card to another player.
func (g *Game) WaitingDelayedHandSourcesFor(viewer uuid.UUID) []DamageSource {
	return g.waitingDelayedCardSources(viewer, core.ZoneHand)
}

// WaitingDelayedExileSources reads saved exile visits in waiting card frames.
// A returned card cannot replace the saved visit.
func (g *Game) WaitingDelayedExileSources() []DamageSource {
	return g.waitingDelayedCardSources(uuid.Nil, core.ZoneExile)
}

// ChooseDamageSourceFromWaitingDelayedCardBindings uses saved hand and exile visits.
// Other source categories need separate enumeration services.
func ChooseDamageSourceFromWaitingDelayedCardBindings(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingDelayedCards: true}
}

// WaitingDelayedGraveyardSources reads saved graveyard visits in waiting card frames.
// A returned card cannot replace the saved visit.
func (g *Game) WaitingDelayedGraveyardSources() []DamageSource {
	return g.waitingDelayedCardSources(uuid.Nil, core.ZoneGraveyard)
}

// ChooseDamageSourceFromWaitingDelayedGraveyardBindings uses saved graveyard visits.
// Other source categories need separate enumeration services.
func ChooseDamageSourceFromWaitingDelayedGraveyardBindings(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingDelayedGraveyard: true}
}
