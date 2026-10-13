package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PutExiledCardsOntoBattlefieldUnderOwnersControl moves saved exile visits together.
// Entry choices cannot substitute a returned card or an absent Aura host.
func PutExiledCardsOntoBattlefieldUnderOwnersControl(subjects ExiledCardSet) ResolvingEffect {
	return &exileEntryEffect{subjects: subjects}
}

type exileEntryEffect struct{ subjects ExiledCardSet }

func (*exileEntryEffect) Text() string {
	return "put exiled cards onto the battlefield under their owners' control"
}
func (*exileEntryEffect) Properties() EffectProperties { return EffectProperties{Mass: true} }
func (e *exileEntryEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{{"subjects", e.subjects}}}
}
func (e *exileEntryEffect) refersToSourcePermanent() bool {
	if set, ok := e.subjects.(linkedExileSet); ok {
		return permanentRefReadsSourcePermanent(set.source)
	}
	return false
}

type preparedExileEntry struct {
	subject   capturedCard
	permanent *Permanent
	host      capturedPermanent
	failed    Card
	owner     uuid.UUID
}

func (e *exileEntryEffect) Apply(ctx *EffectContext) error {
	g := ctx.Game
	savedX := g.resolution.X()
	g.resolution.SetX(0)
	defer g.resolution.SetX(savedX)
	byOwner := make(map[uuid.UUID][]capturedCard)
	seen := make(map[uuid.UUID]bool)
	for _, subject := range e.subjects.resolveExiledCards(ctx) {
		if subject.zone != core.ZoneExile || subject.visit == uuid.Nil || seen[subject.id] {
			continue
		}
		exiled := g.FindExiledCard(subject.id)
		if exiled == nil || exiled.Visit != subject.visit || exiled.Card.IsToken() || !IsPermanentCard.Match(exiled.Card) || g.GetPlayer(exiled.Card.Owner()) == nil {
			continue
		}
		seen[subject.id] = true
		byOwner[exiled.Card.Owner()] = append(byOwner[exiled.Card.Owner()], subject)
	}
	prepared := make([]preparedExileEntry, 0, len(seen))
	for _, owner := range []Player{g.ActivePlayerObj(), g.NonActivePlayerObj()} {
		cards := make([]Card, 0, len(byOwner[owner.PlayerID()]))
		visits := make(map[uuid.UUID]capturedCard)
		for _, subject := range byOwner[owner.PlayerID()] {
			if exiled := g.FindExiledCard(subject.id); exiled != nil && exiled.Visit == subject.visit && exiled.Card.Owner() == owner.PlayerID() {
				cards = append(cards, exiled.Card)
				visits[subject.id] = subject
			}
		}
		for _, selected := range g.orderCardsByChoice(owner, cards, "choose the next card for entry choices") {
			subject := visits[selected.ID()]
			exiled := g.FindExiledCard(subject.id)
			if exiled == nil || exiled.Visit != subject.visit || exiled.Card.Owner() != owner.PlayerID() {
				continue
			}
			card := exiled.Card.Copy()
			card.SetID(subject.id)
			perm, failed := g.prepareBattlefieldEntryResultWithContext(card, owner.PlayerID(), nil, battlefieldEntryContext{}, false)
			if perm == nil {
				if failed != nil {
					prepared = append(prepared, preparedExileEntry{subject: subject, failed: failed, owner: owner.PlayerID()})
				}
				continue
			}
			host, ok := g.chooseExileEntryAuraHost(perm, owner)
			if !ok {
				continue
			}
			prepared = append(prepared, preparedExileEntry{subject: subject, permanent: perm, host: host, owner: owner.PlayerID()})
		}
	}
	byGraveyardOwner := make(map[uuid.UUID][]Card)
	for _, entry := range prepared {
		if entry.failed != nil {
			byGraveyardOwner[entry.owner] = append(byGraveyardOwner[entry.owner], entry.failed)
		}
	}
	for _, owner := range []Player{g.ActivePlayerObj(), g.NonActivePlayerObj()} {
		byGraveyardOwner[owner.PlayerID()] = g.orderCardsForGraveyard(owner, byGraveyardOwner[owner.PlayerID()])
	}
	failedVisits := make(map[uuid.UUID]capturedCard)
	entered := make([]preparedExileEntry, 0, len(prepared))
	for _, entry := range prepared {
		exiled := g.FindExiledCard(entry.subject.id)
		if exiled == nil || exiled.Visit != entry.subject.visit || exiled.Card.Owner() != entry.owner {
			continue
		}
		if entry.failed != nil {
			failedVisits[entry.subject.id] = entry.subject
			continue
		}
		if entry.host.id != uuid.Nil {
			host := g.FindPermanent(entry.host.id)
			if host == nil || host.PhasedOut || host.incarnationID != entry.host.incarnation || !g.canAttach(entry.permanent, host) {
				continue
			}
		}
		if g.isLandEntryProhibited(entry.permanent.HasType(core.TypeLand)) {
			continue
		}
		if _, ok := g.RemoveFromExile(entry.subject.id); !ok {
			continue
		}
		entered = append(entered, entry)
	}
	var failures []core.GameEvent
	for _, owner := range g.players {
		for _, card := range byGraveyardOwner[owner.PlayerID()] {
			subject, ok := failedVisits[card.ID()]
			if !ok {
				continue
			}
			exiled := g.FindExiledCard(card.ID())
			if exiled == nil || exiled.Visit != subject.visit || exiled.Card.Owner() != owner.PlayerID() {
				continue
			}
			if _, ok := g.RemoveFromExile(card.ID()); !ok {
				continue
			}
			owner.AddToGraveyard(card)
			failures = append(failures, core.GameEvent{Type: core.EvtZoneChange, SourceID: card.ID(), PlayerID: owner.PlayerID(), ActorID: ctx.Controller, FromZone: core.ZoneExile, ToZone: core.ZoneGraveyard})
		}
	}
	for _, entry := range entered {
		g.registerBattlefieldEntry(entry.permanent, nil, uuid.Nil)
	}
	for _, entry := range entered {
		if entry.host.id != uuid.Nil {
			g.Attach(entry.permanent.ID(), entry.host.id)
		}
	}
	for _, event := range failures {
		g.FireEvent(event)
	}
	for _, entry := range entered {
		g.completeBattlefieldEntryFrom(entry.permanent, entry.permanent.ControllerID(), ctx.Controller, 0, core.ZoneExile)
	}
	return nil
}

func (g *Game) chooseExileEntryAuraHost(perm *Permanent, owner Player) (capturedPermanent, bool) {
	if !perm.HasSubType("Aura") {
		return capturedPermanent{}, true
	}
	var candidates []*Permanent
	for _, host := range g.FilterBattlefield(AnyPermanent) {
		if !host.PhasedOut && g.canAttach(perm, host) {
			candidates = append(candidates, host)
		}
	}
	if len(candidates) == 0 {
		return capturedPermanent{}, false
	}
	saved := make(map[uuid.UUID]capturedPermanent, len(candidates))
	for _, host := range candidates {
		saved[host.ID()] = capturedPermanent{host.ID(), host.incarnationID}
	}
	chosen := owner.ChoosePermanent(candidates, "Choose what the entering Aura enchants", g)
	subject := saved[candidates[0].ID()]
	if chosen != nil {
		if offered, ok := saved[chosen.ID()]; ok {
			subject = offered
		}
	}
	host := g.FindPermanent(subject.id)
	if host == nil || host.PhasedOut || host.incarnationID != subject.incarnation || !g.canAttach(perm, host) {
		return capturedPermanent{}, false
	}
	return subject, true
}
