package mage

import (
	"errors"
	"fmt"
	"sort"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// OwnershipChange describes the net ownership of a card transferred during
// an ante game.
type OwnershipChange struct {
	CardID        uuid.UUID
	CardName      string
	OriginalOwner uuid.UUID
	FinalOwner    uuid.UUID
}

// NewGameWithAnte creates an ante-enabled game and moves the explicitly
// selected cards from each player's library into the shared ante zone.
func NewGameWithAnte(playerA, playerB Player, anteA, anteB []Card) (*Game, error) {
	if playerA == nil || playerB == nil {
		return nil, ErrPlayerNotFound
	}
	seen := make(map[uuid.UUID]struct{}, len(anteA)+len(anteB))
	validate := func(player Player, selected []Card) error {
		library := make(map[uuid.UUID]Card, len(player.Library()))
		for _, c := range player.Library() {
			if c != nil {
				library[c.ID()] = c
			}
		}
		for _, supplied := range selected {
			if supplied == nil {
				return errors.New("ante card is nil")
			}
			if _, ok := seen[supplied.ID()]; ok {
				return fmt.Errorf("duplicate ante card %s", supplied.ID())
			}
			seen[supplied.ID()] = struct{}{}
			actual, ok := library[supplied.ID()]
			if !ok {
				return fmt.Errorf("ante card %s is not in %s's library", supplied.Name(), player.Name())
			}
			if actual.Owner() != player.PlayerID() {
				return fmt.Errorf("ante card %s is not owned by %s", actual.Name(), player.Name())
			}
			if actual.IsToken() {
				return fmt.Errorf("token %s cannot be anted", actual.Name())
			}
		}
		return nil
	}
	if err := validate(playerA, anteA); err != nil {
		return nil, err
	}
	if err := validate(playerB, anteB); err != nil {
		return nil, err
	}

	g := newGame(playerA, playerB, true)
	moveInitial := func(player Player, selected []Card) {
		ids := make(map[uuid.UUID]struct{}, len(selected))
		for _, c := range selected {
			ids[c.ID()] = struct{}{}
		}
		kept := make([]Card, 0, len(player.Library())-len(selected))
		for _, c := range player.Library() {
			if _, ok := ids[c.ID()]; ok {
				player.AddToAnte(c)
			} else {
				kept = append(kept, c)
			}
		}
		player.SetLibrary(kept)
	}
	moveInitial(playerA, anteA)
	moveInitial(playerB, anteB)
	return g, nil
}

// AnteEnabled reports whether this game was created with ante enabled.
func (g *Game) AnteEnabled() bool { return g != nil && g.anteEnabled }

// AnteCards lists every card in the shared, public ante zone.
func (g *Game) AnteCards() ([]Card, error) {
	if g == nil || !g.anteEnabled {
		return nil, ErrAnteDisabled
	}
	var cards []Card
	for _, player := range g.players {
		cards = append(cards, player.Ante()...)
	}
	return append([]Card(nil), cards...), nil
}

// AnteCardsOwnedBy lists ante cards whose current owner is ownerID.
func (g *Game) AnteCardsOwnedBy(ownerID uuid.UUID) ([]Card, error) {
	if g == nil || !g.anteEnabled {
		return nil, ErrAnteDisabled
	}
	if g.GetPlayer(ownerID) == nil {
		return nil, ErrPlayerNotFound
	}
	cards, _ := g.AnteCards()
	owned := make([]Card, 0, len(cards))
	for _, c := range cards {
		if c.Owner() == ownerID {
			owned = append(owned, c)
		}
	}
	return owned, nil
}

// CardZone returns the modeled zone containing cardID, or ZoneAny when the
// card is absent. For shared zones it does not imply a controller or owner.
func (g *Game) CardZone(cardID uuid.UUID) Zone { return g.zoneOfTarget(cardID) }

// MoveToAnte antes a nontoken card currently owned by ownerID, preserving its
// identity and firing a zone-change event.
func (g *Game) MoveToAnte(ownerID, cardID uuid.UUID) error {
	if !g.anteEnabled {
		return ErrAnteDisabled
	}
	if g.anteSettled {
		return ErrAnteSettled
	}
	owner := g.GetPlayer(ownerID)
	if owner == nil {
		return ErrPlayerNotFound
	}
	card := g.FindCardAnywhere(cardID)
	if card == nil {
		return fmt.Errorf("card not found: %s", cardID)
	}
	if card.Owner() != ownerID {
		return errors.New("only a card's owner can ante it")
	}
	if card.IsToken() {
		return errors.New("tokens cannot be anted")
	}
	from := g.CardZone(cardID)
	if from == ZoneAny || from == ZoneAnte {
		return fmt.Errorf("card cannot be anted from %s", from)
	}
	removed, playerID, selfAbilities, err := g.removeCardForAnte(cardID, from)
	if err != nil {
		return err
	}
	owner.AddToAnte(removed)
	evt := GameEvent{Type: EvtZoneChange, SourceID: cardID, PlayerID: playerID, FromZone: from, ToZone: ZoneAnte}
	g.FireEvent(evt)
	if from == ZoneBattlefield {
		g.checkAbilitiesForEvent(selfAbilities, &evt, cardID, playerID)
	}
	return nil
}

func (g *Game) removeCardForAnte(cardID uuid.UUID, from Zone) (Card, uuid.UUID, []Ability, error) {
	switch from {
	case ZoneBattlefield:
		perm := g.FindPermanent(cardID)
		if perm == nil || perm.IsToken {
			return nil, uuid.Nil, nil, ErrPermanentNotFound
		}
		controller := perm.ControllerID()
		card := perm.Card
		g.RemoveFromBattlefield(perm)
		return card, controller, g.LKIAbilities(cardID), nil
	case ZoneExile:
		card, ok := g.RemoveFromExile(cardID)
		if !ok {
			return nil, uuid.Nil, nil, errors.New("card not found in exile")
		}
		return card, card.Owner(), nil, nil
	case ZoneStack:
		for i, obj := range g.stack.objects {
			if obj.Card != nil && obj.Card.ID() == cardID {
				g.stack.objects = append(append([]*StackObject(nil), g.stack.objects[:i]...), g.stack.objects[i+1:]...)
				return obj.Card, obj.Controller, nil, nil
			}
		}
	case ZoneHand, ZoneLibrary, ZoneGraveyard:
		for _, player := range g.players {
			switch from {
			case ZoneHand:
				if c, ok := player.RemoveFromHand(cardID); ok {
					return c, player.PlayerID(), nil, nil
				}
			case ZoneLibrary:
				for i, c := range player.Library() {
					if c.ID() == cardID {
						lib := append([]Card(nil), player.Library()...)
						player.SetLibrary(append(lib[:i], lib[i+1:]...))
						return c, player.PlayerID(), nil, nil
					}
				}
			case ZoneGraveyard:
				if c, ok := player.RemoveFromGraveyard(cardID); ok {
					g.FireEvent(GameEvent{Type: EvtCardsLeftGraveyard, PlayerID: player.PlayerID(), Amount: 1})
					return c, player.PlayerID(), nil, nil
				}
			}
		}
	}
	return nil, uuid.Nil, nil, errors.New("card not found in expected zone")
}

// RemoveFromAnte removes cardID from the shared ante zone. If a destination
// is supplied, the emitted zone-change event records it; otherwise ZoneAny is
// used and the caller remains responsible for placing the returned card.
func (g *Game) RemoveFromAnte(cardID uuid.UUID, destination ...Zone) (Card, error) {
	if !g.anteEnabled {
		return nil, ErrAnteDisabled
	}
	if g.anteSettled {
		return nil, ErrAnteSettled
	}
	to := ZoneAny
	if len(destination) > 0 {
		to = destination[0]
	}
	for _, player := range g.players {
		if card, ok := player.RemoveFromAnte(cardID); ok {
			g.FireEvent(GameEvent{Type: EvtZoneChange, SourceID: cardID, PlayerID: player.PlayerID(), FromZone: ZoneAnte, ToZone: to})
			return card, nil
		}
	}
	return nil, errors.New("card not found in ante")
}

// ChangeOwner changes a card's owner without changing its identity, zone, or
// controller. The card object is replaced copy-on-write for clone isolation.
func (g *Game) ChangeOwner(cardID, newOwnerID uuid.UUID) error {
	if !g.anteEnabled {
		return ErrAnteDisabled
	}
	if g.anteSettled {
		return ErrAnteSettled
	}
	if g.GetPlayer(newOwnerID) == nil {
		return ErrPlayerNotFound
	}
	card := g.FindCardAnywhere(cardID)
	if card == nil {
		return fmt.Errorf("card not found: %s", cardID)
	}
	if card.IsToken() {
		return errors.New("tokens cannot change ownership")
	}
	if card.Owner() == newOwnerID {
		return nil
	}
	if _, ok := g.originalOwners[cardID]; !ok {
		g.originalOwners[cardID] = card.Owner()
	}
	replacement := card.Copy()
	replacement.SetID(cardID)
	replacement.SetOwner(newOwnerID)
	if !g.replaceCardReference(cardID, replacement) {
		return fmt.Errorf("card not found: %s", cardID)
	}
	return nil
}

func replaceCardInSlice(cards []Card, cardID uuid.UUID, replacement Card) ([]Card, bool) {
	for i, c := range cards {
		if c.ID() == cardID {
			out := append([]Card(nil), cards...)
			out[i] = replacement
			return out, true
		}
	}
	return cards, false
}

func (g *Game) replaceCardReference(cardID uuid.UUID, replacement Card) bool {
	if perm := g.FindPermanentIncludingPhased(cardID); perm != nil {
		mutable := g.MutablePermanent(cardID)
		mutable.Card = replacement
		return true
	}
	if ep := g.zones.EnteringPermanent(); ep != nil && ep.ID() == cardID {
		ep.Card = replacement
		return true
	}
	for _, obj := range g.stack.objects {
		if obj.Card != nil && obj.Card.ID() == cardID {
			obj.Card = replacement
			return true
		}
	}
	if rc := g.resolution.ResolvingCard(); rc != nil && rc.ID() == cardID {
		g.resolution.SetResolvingCard(replacement)
		return true
	}
	if g.zones.ReplaceExiledCard(cardID, replacement) {
		return true
	}
	for _, player := range g.players {
		if cards, ok := replaceCardInSlice(player.Library(), cardID, replacement); ok {
			player.SetLibrary(cards)
			return true
		}
		if cards, ok := replaceCardInSlice(player.Hand(), cardID, replacement); ok {
			player.SetHand(cards)
			return true
		}
		if cards, ok := replaceCardInSlice(player.Graveyard(), cardID, replacement); ok {
			player.SetGraveyard(cards)
			return true
		}
		if cards, ok := replaceCardInSlice(player.Ante(), cardID, replacement); ok {
			player.SetAnte(cards)
			return true
		}
	}
	return false
}

// AnteResult settles the ante to the sole winner and returns all net ownership
// changes from original owner to final owner.
func (g *Game) AnteResult() ([]OwnershipChange, error) {
	if !g.anteEnabled {
		return nil, ErrAnteDisabled
	}
	if g.anteSettled {
		return append([]OwnershipChange(nil), g.anteResult...), nil
	}
	if !g.IsGameOver() {
		return nil, errors.New("game is not finished")
	}
	var winner Player
	for _, player := range g.players {
		if player.IsAlive() {
			if winner != nil {
				return nil, errors.New("game does not have exactly one winner")
			}
			winner = player
		}
	}
	if winner == nil {
		return nil, errors.New("game does not have exactly one winner")
	}
	cards, _ := g.AnteCards()
	for _, card := range cards {
		if err := g.ChangeOwner(card.ID(), winner.PlayerID()); err != nil {
			return nil, err
		}
	}
	result := make([]OwnershipChange, 0, len(g.originalOwners))
	for cardID, original := range g.originalOwners {
		card := g.FindCardAnywhere(cardID)
		if card == nil || card.Owner() == original {
			continue
		}
		result = append(result, OwnershipChange{CardID: cardID, CardName: card.Name(), OriginalOwner: original, FinalOwner: card.Owner()})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CardName == result[j].CardName {
			return result[i].CardID.String() < result[j].CardID.String()
		}
		return result[i].CardName < result[j].CardName
	})
	g.anteResult = append([]OwnershipChange(nil), result...)
	g.anteSettled = true
	return append([]OwnershipChange(nil), result...), nil
}
