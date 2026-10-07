package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type optionalHandEntry struct {
	player PlayerRef
	filter CardFilter
}

// MayPutFromHandOntoBattlefield offers one matching permanent card at resolution.
// The saved hand visit prevents a returned card from replacing the chosen object.
func MayPutFromHandOntoBattlefield(player PlayerRef, filter CardFilter) ResolvingEffect {
	return &optionalHandEntry{player: player, filter: filter}
}
func (*optionalHandEntry) Text() string                 { return "may put a card from hand onto the battlefield" }
func (*optionalHandEntry) Properties() EffectProperties { return EffectProperties{} }
func (e *optionalHandEntry) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"player", e.player}}}
}
func (e *optionalHandEntry) Apply(ctx *EffectContext) error {
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return nil
	}
	visitor, ok := player.(interface{ handVisit(uuid.UUID) uuid.UUID })
	if !ok {
		return nil
	}
	var candidates []Card
	visits := make(map[uuid.UUID]uuid.UUID)
	for _, card := range player.Hand() {
		if IsPermanentCard.Match(card) && e.filter.Match(card) {
			candidates = append(candidates, card)
			visits[card.ID()] = visitor.handVisit(card.ID())
		}
	}
	if len(candidates) == 0 || !player.ChooseMayAbility(e.Text()) {
		return nil
	}
	chosen := player.ChooseCardFromHand(candidates, e.Text(), ctx.Game)
	if chosen == nil {
		return nil
	}
	visit, offered := visits[chosen.ID()]
	if !offered || visit == uuid.Nil || visitor.handVisit(chosen.ID()) != visit {
		return nil
	}
	var current Card
	for _, card := range player.Hand() {
		if card.ID() == chosen.ID() {
			current = card
			break
		}
	}
	if current == nil || !IsPermanentCard.Match(current) || !e.filter.Match(current) {
		return nil
	}
	return ctx.Game.putChosenHandCard(player, current, visit)
}

func (g *Game) putChosenHandCard(player Player, card Card, visit uuid.UUID) error {
	if card.HasType(core.TypeLand) && g.effects.Rules.LandsCantEnter() {
		return nil
	}
	var host *Permanent
	if card.HasSubType("Aura") {
		source := NewPermanent(card, player.PlayerID())
		var hosts []*Permanent
		for _, p := range g.FilterBattlefield(AnyPermanent) {
			if !p.PhasedOut && g.canAttach(source, p) {
				hosts = append(hosts, p)
			}
		}
		if len(hosts) == 0 {
			return nil
		}
		chosen := player.ChoosePermanent(hosts, "enchant", g)
		selected := hosts[0]
		if chosen != nil {
			for _, candidate := range hosts {
				if candidate.ID() == chosen.ID() {
					selected = candidate
					break
				}
			}
		}
		host = g.FindPermanent(selected.ID())
		if host == nil || host.incarnationID != selected.incarnationID || !g.canAttach(source, host) {
			return nil
		}

	}
	visitor, ok := player.(interface{ handVisit(uuid.UUID) uuid.UUID })
	if !ok || visitor.handVisit(card.ID()) != visit {
		return nil
	}
	removed, ok := player.RemoveFromHand(card.ID())
	if !ok {
		return nil
	}
	perm := g.prepareBattlefieldEntry(removed, player.PlayerID(), nil, 0, false)
	if perm == nil {
		return nil
	}
	g.registerBattlefieldEntry(perm, nil, uuid.Nil)
	if host != nil {
		g.Attach(perm.ID(), host.ID())
	}
	g.completeBattlefieldEntry(perm, player.PlayerID(), player.PlayerID(), 0)
	return nil
}
