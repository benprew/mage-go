package mage

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// MaySearchLibraryToHand offers a bounded search, reveal and hand movement.
// Saved library visits prevent returned cards from replacing chosen cards.
func MaySearchLibraryToHand(player PlayerRef, maximum int, filter CardFilter) ResolvingEffect {
	return &boundedLibrarySearchEffect{player: player, maximum: maximum, filter: filter}
}

type boundedLibrarySearchEffect struct {
	player  PlayerRef
	maximum int
	filter  CardFilter
}

func (e *boundedLibrarySearchEffect) Text() string {
	return fmt.Sprintf("may search your library for up to %d matching cards, reveal them, put them into your hand, then shuffle", e.maximum)
}
func (*boundedLibrarySearchEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *boundedLibrarySearchEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"player", e.player}}}
	if e.maximum < 0 {
		n.problem = "supply a nonnegative search maximum"
	}
	return n
}
func (e *boundedLibrarySearchEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}
func (e *boundedLibrarySearchEffect) Apply(ctx *EffectContext) error {
	player := handOperationPlayer(e.player, ctx)
	if player == nil || !player.ChooseMayAbility(e.Text()) {
		return nil
	}
	visitor, ok := player.(interface{ libraryVisit(uuid.UUID) uuid.UUID })
	if !ok {
		return errors.New("library search requires an engine library-visit identity")
	}
	visits := make(map[uuid.UUID]uuid.UUID)
	order := make([]uuid.UUID, 0, len(player.Library()))
	for _, card := range player.Library() {
		if e.filter.Match(card) && visitor.libraryVisit(card.ID()) != uuid.Nil {
			visits[card.ID()] = visitor.libraryVisit(card.ID())
			order = append(order, card.ID())
		}
	}
	selected := make(map[uuid.UUID]bool)
	chosenIDs := make([]uuid.UUID, 0, min(max(e.maximum, 0), len(order)))
	currentCard := func(id uuid.UUID) Card {
		if visitor.libraryVisit(id) != visits[id] {
			return nil
		}
		for _, card := range player.Library() {
			if card.ID() == id && e.filter.Match(card) {
				return card
			}
		}
		return nil
	}
	for range min(max(e.maximum, 0), len(order)) {
		candidates := make([]Card, 0, len(order))
		for _, id := range order {
			if !selected[id] {
				if card := currentCard(id); card != nil {
					candidates = append(candidates, card)
				}
			}
		}
		chosen := chooseLibrarySearchCard(player, candidates, e.Text(), ctx.Game, false)
		if chosen == nil || currentCard(chosen.ID()) == nil {
			break
		}
		selected[chosen.ID()] = true
		chosenIDs = append(chosenIDs, chosen.ID())
	}
	found := make([]Card, 0, len(chosenIDs))
	for _, id := range chosenIDs {
		if card := currentCard(id); card != nil {
			if ctx.Game.GetPlayer(card.Owner()) == nil {
				return ErrPlayerNotFound
			}
			found = append(found, card)
		}
	}
	moved := make(map[uuid.UUID]bool, len(found))
	for _, card := range found {
		ctx.Game.recordPublicReveal(card)
		moved[card.ID()] = true
	}
	remaining := make([]Card, 0, len(player.Library())-len(found))
	for _, card := range player.Library() {
		if !moved[card.ID()] {
			remaining = append(remaining, card)
		}
	}
	player.SetLibrary(remaining)
	for _, card := range found {
		ctx.Game.GetPlayer(card.Owner()).AddToHand(card)
	}
	for _, card := range found {
		ctx.Game.FireEvent(core.GameEvent{Type: core.EvtZoneChange, SourceID: card.ID(), PlayerID: card.Owner(), FromZone: core.ZoneLibrary, ToZone: core.ZoneHand})
	}
	player.ShuffleLibrary()
	return nil
}
