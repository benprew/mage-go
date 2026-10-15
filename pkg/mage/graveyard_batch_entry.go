package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PutGraveyardCreaturesOntoBattlefieldUnderOwnersControl returns creatures together.
// Exact visits prevent entry choices from moving a returned card.
func PutGraveyardCreaturesOntoBattlefieldUnderOwnersControl() ResolvingEffect {
	return &graveyardCreatureEntryEffect{}
}

type graveyardCreatureEntryEffect struct{}

func (*graveyardCreatureEntryEffect) Text() string {
	return "return all graveyard creatures under their owners' control"
}
func (*graveyardCreatureEntryEffect) Properties() EffectProperties {
	return EffectProperties{Mass: true}
}
func (*graveyardCreatureEntryEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect"}
}

type preparedGraveyardEntry struct {
	subject   capturedCard
	owner     uuid.UUID
	permanent *Permanent
	host      capturedPermanent
}

func (*graveyardCreatureEntryEffect) Apply(ctx *EffectContext) error {
	g := ctx.Game
	byOwner := make(map[uuid.UUID][]capturedCard)
	seen := make(map[uuid.UUID]bool)
	for _, player := range g.players {
		for _, card := range player.Graveyard() {
			visit := g.graveyardVisit(card.ID())
			if visit == uuid.Nil || seen[card.ID()] || card.IsToken() || !card.HasType(core.TypeCreature) || card.Owner() != player.PlayerID() {
				continue
			}
			seen[card.ID()] = true
			byOwner[player.PlayerID()] = append(byOwner[player.PlayerID()], capturedCard{id: card.ID(), visit: visit, zone: core.ZoneGraveyard})
		}
	}
	return g.enterSavedGraveyardCards(ctx, byOwner)
}

func (g *Game) enterSavedGraveyardCards(ctx *EffectContext, byOwner map[uuid.UUID][]capturedCard) error {
	savedX := g.resolution.X()
	g.resolution.SetX(0)
	defer g.resolution.SetX(savedX)
	prepared := make([]preparedGraveyardEntry, 0)
	for _, owner := range []Player{g.ActivePlayerObj(), g.NonActivePlayerObj()} {
		var cards []Card
		visits := make(map[uuid.UUID]capturedCard)
		for _, subject := range byOwner[owner.PlayerID()] {
			if card := g.graveyardEntryCard(owner, subject); card != nil {
				cards = append(cards, card)
				visits[card.ID()] = subject
			}
		}
		for _, selected := range g.orderCardsByChoice(owner, cards, "choose the next card for entry choices") {
			subject := visits[selected.ID()]
			current := g.graveyardEntryCard(owner, subject)
			if current == nil {
				continue
			}
			card := current.Copy()
			card.SetID(subject.id)
			perm, _ := g.prepareBattlefieldEntryResultWithContext(card, owner.PlayerID(), nil, battlefieldEntryContext{actor: ctx.Controller}, false)
			if perm == nil {
				continue
			}
			host, ok := g.chooseEntryAuraHost(perm, owner)
			if !ok {
				continue
			}
			prepared = append(prepared, preparedGraveyardEntry{subject: subject, owner: owner.PlayerID(), permanent: perm, host: host})
		}
	}
	entered := make([]preparedGraveyardEntry, 0, len(prepared))
	for _, entry := range prepared {
		owner := g.GetPlayer(entry.owner)
		if owner == nil || g.graveyardEntryCard(owner, entry.subject) == nil || g.isLandEntryProhibited(entry.permanent.HasType(core.TypeLand)) {
			continue
		}
		if entry.host.id != uuid.Nil {
			host := g.FindPermanent(entry.host.id)
			if host == nil || host.PhasedOut || host.incarnationID != entry.host.incarnation || !g.canAttachEntering(entry.permanent, host) {
				continue
			}
		}
		if _, ok := owner.RemoveFromGraveyard(entry.subject.id); !ok {
			continue
		}
		entered = append(entered, entry)
	}
	for _, entry := range entered {
		g.registerBattlefieldEntry(entry.permanent, nil, uuid.Nil)
	}
	for _, entry := range entered {
		if entry.host.id != uuid.Nil {
			g.Attach(entry.permanent.ID(), entry.host.id)
		}
	}
	g.effects.Apply(g)
	left := make(map[uuid.UUID]int)
	for _, entry := range entered {
		left[entry.owner]++
	}
	for _, player := range g.players {
		if count := left[player.PlayerID()]; count > 0 {
			g.FireEvent(core.GameEvent{Type: core.EvtCardsLeftGraveyard, PlayerID: player.PlayerID(), Amount: count})
		}
	}
	for _, entry := range entered {
		g.observeBattlefieldEntryFrom(entry.permanent, entry.owner, ctx.Controller, 0, core.ZoneGraveyard)
	}
	for _, entry := range entered {
		g.finishRetainedEntry(entry.permanent)
	}
	return nil
}

func (g *Game) graveyardEntryCard(owner Player, subject capturedCard) Card {
	if subject.visit == uuid.Nil || g.graveyardVisit(subject.id) != subject.visit {
		return nil
	}
	for _, card := range owner.Graveyard() {
		if card.ID() == subject.id && card.Owner() == owner.PlayerID() {
			return card
		}
	}
	return nil
}
