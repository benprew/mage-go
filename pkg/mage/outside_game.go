package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type outsideGameCard struct {
	card  Card
	visit uuid.UUID
}

// SetOutsideGameCards sets the owned cards available from outside this game.
// The caller supplies the pool permitted by the match or casual game.
// Invalid input leaves the previous pool unchanged.
func (g *Game) SetOutsideGameCards(player uuid.UUID, cards []Card) error {
	if g.GetPlayer(player) == nil {
		return ErrPlayerNotFound
	}
	seen := make(map[uuid.UUID]bool, len(cards))
	entries := make([]outsideGameCard, 0, len(cards))
	for _, card := range cards {
		if card == nil || card.ID() == uuid.Nil || card.IsToken() || len(card.Types()) == 0 || card.Owner() != player {
			return fmt.Errorf("supply owned traditional cards for the outside-game pool")
		}
		if seen[card.ID()] || g.FindCardAnywhere(card.ID()) != nil {
			return fmt.Errorf("card %s is repeated or already in this game", card.Name())
		}
		for owner, pool := range g.outsideGame {
			if owner == player {
				continue
			}
			for _, other := range pool {
				if other.card.ID() == card.ID() {
					return fmt.Errorf("card %s is already in another outside-game pool", card.Name())
				}
			}
		}
		seen[card.ID()] = true
		entries = append(entries, outsideGameCard{cloneZoneCards([]Card{card})[0], uuid.New()})
	}
	if g.outsideGame == nil {
		g.outsideGame = make(map[uuid.UUID][]outsideGameCard)
	}
	g.outsideGame[player] = entries
	return nil
}

// OutsideGameCards returns separate card snapshots from the supplied pool.
// These cards are not in any game zone and are not public game information.
func (g *Game) OutsideGameCards(player uuid.UUID) []Card {
	pool := g.outsideGame[player]
	cards := make([]Card, len(pool))
	for i, entry := range pool {
		cards[i] = entry.card
	}
	return cloneZoneCards(cards)
}
func cloneOutsideGame(pools map[uuid.UUID][]outsideGameCard) map[uuid.UUID][]outsideGameCard {
	if pools == nil {
		return nil
	}
	result := make(map[uuid.UUID][]outsideGameCard, len(pools))
	for player, pool := range pools {
		entries := slices.Clone(pool)
		for i := range entries {
			entries[i].card = cloneZoneCards([]Card{entries[i].card})[0]
		}
		result[player] = entries
	}
	return result
}
func (g *Game) outsideGameEligible(entry outsideGameCard, player uuid.UUID, filter CardFilter) bool {
	card := entry.card
	if card.Owner() != player || card.IsToken() || len(card.Types()) == 0 || g.FindCardAnywhere(card.ID()) != nil || !filter.Match(card) {
		return false
	}
	if restricted, ok := card.(interface{ RequiresAnte() bool }); ok && restricted.RequiresAnte() && !g.AnteEnabled() {
		return false
	}
	return true
}

// PutOutsideGameCardInHand brings one matching owned card into a player's hand.
// A required choice uses the first candidate if the answer is invalid.
func PutOutsideGameCardInHand(player PlayerRef, filter CardFilter) ResolvingEffect {
	return &outsideGameEntry{player, filter}
}

type outsideGameEntry struct {
	player PlayerRef
	filter CardFilter
}

func (*outsideGameEntry) Text() string { return "put an owned card from outside the game into hand" }
func (*outsideGameEntry) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *outsideGameEntry) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}
func (e *outsideGameEntry) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{{"player", e.player}}}
}
func (e *outsideGameEntry) Apply(ctx *EffectContext) error {
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	g := ctx.Game
	player := g.GetPlayer(id)
	if player == nil {
		return nil
	}
	var candidates []Card
	entries := make(map[uuid.UUID]outsideGameCard)
	for _, entry := range g.outsideGame[id] {
		if g.outsideGameEligible(entry, id, e.filter) {
			candidates = append(candidates, entry.card)
			entries[entry.card.ID()] = entry
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	selected := candidates[0].ID()
	answer := player.ChooseCardFromOutsideGame(cloneZoneCards(candidates), e.Text(), g)
	if answer != nil {
		if _, found := entries[answer.ID()]; found {
			selected = answer.ID()
		}
	}
	saved := entries[selected]
	proposal := &OutsideGameEntryAction{source: ctx.SourceID, player: id, card: selected, visit: saved.visit}
	result, ok := g.effects.ApplyReplacements(proposal, g).(*OutsideGameEntryAction)
	if !ok || result == nil {
		return nil
	}
	// A replacement cannot extend the permission to a different card or player.
	if result.card != proposal.card || result.player != id || result.visit != saved.visit {
		return nil
	}
	for i, current := range g.outsideGame[id] {
		if current.card.ID() != selected || current.visit != saved.visit {
			continue
		}
		if !g.outsideGameEligible(current, id, e.filter) {
			return nil
		}
		g.outsideGame[id] = slices.Delete(g.outsideGame[id], i, i+1)
		player.AddToHand(current.card)
		g.FireEvent(core.GameEvent{Type: core.EvtZoneChange, SourceID: selected, PlayerID: id, FromZone: core.ZoneAny, ToZone: core.ZoneHand})
		return nil
	}
	return nil
}

// OutsideGameEntryAction proposes entry from an absent origin into hand.
// Outside the game is not a zone. ZoneAny marks the absent event origin only.
type OutsideGameEntryAction struct{ source, player, card, visit uuid.UUID }

func (a *OutsideGameEntryAction) ActionSource() uuid.UUID { return a.source }
func (a *OutsideGameEntryAction) PlayerID() uuid.UUID     { return a.player }
func (a *OutsideGameEntryAction) CardID() uuid.UUID       { return a.card }

// ChooseCardFromOutsideGame selects one card from the supplied candidates.
func (p *BasePlayer) ChooseCardFromOutsideGame(cards []Card, _ string, _ GameReader) Card {
	if len(cards) == 0 {
		return nil
	}
	return cards[0]
}
