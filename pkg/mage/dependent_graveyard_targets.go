package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// GraveyardCardTargetsBinding keeps a collection of exact graveyard visits.
type GraveyardCardTargetsBinding struct{ *targetDeclaration }

// DeclareGraveyardCardTargets permits any number of cards in an earlier player's graveyard.
func DeclareGraveyardCardTargets(name string, player *PlayerTargetBinding, filters ...CardFilter) *GraveyardCardTargetsBinding {
	spec := &playerGraveyardTargets{BaseTarget: BaseTarget{min: 0, max: int(^uint(0) >> 1)}, player: player, filters: slices.Clone(filters)}
	return &GraveyardCardTargetsBinding{&targetDeclaration{name: name, spec: spec}}
}
func (*GraveyardCardTargetsBinding) cardSet() {}
func (b *GraveyardCardTargetsBinding) resolveCards(ctx *EffectContext) []Card {
	var cards []Card
	for _, value := range ctx.targetBindings[b.targetDeclaration] {
		if value.incarnation == uuid.Nil || ctx.Game.graveyardVisit(value.id) != value.incarnation {
			continue
		}
		if card := ctx.Game.FindCardAnywhere(value.id); card != nil {
			cards = append(cards, card)
		}
	}
	return cards
}
func (b *GraveyardCardTargetsBinding) inspectDefinition() definitionNode {
	return targetReadNode(b.targetDeclaration, "card-set")
}

type playerGraveyardTargets struct {
	BaseTarget
	player  *PlayerTargetBinding
	filters []CardFilter
}

func (t *playerGraveyardTargets) Possible(controller uuid.UUID, _ Card, g *Game) []uuid.UUID {
	if t.player == nil {
		return nil
	}
	id, ok := t.player.resolvePlayer(&EffectContext{Game: g, Controller: controller, targetBindings: g.resolution.targetQueryBindings})
	if !ok {
		return nil
	}
	var ids []uuid.UUID
	for _, card := range g.GetPlayer(id).Graveyard() {
		if !slices.ContainsFunc(t.filters, func(filter CardFilter) bool { return !filter.Match(card) }) {
			ids = append(ids, card.ID())
		}
	}
	return ids
}
func (t *playerGraveyardTargets) Choose(_ uuid.UUID, _ Card, _ *Game, _ []uuid.UUID) error {
	return nil
}
func (t *playerGraveyardTargets) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-graveyard-player", children: []definitionEdge{{"player", t.player}}}
}

// ReturnGraveyardCardsToLibraryTop moves saved cards together, then sends events.
// The selected player chooses their order from top to bottom.
func ReturnGraveyardCardsToLibraryTop(cards CardSet, chooser PlayerRef) ResolvingEffect {
	return &graveyardCardsToTop{cards: cards, chooser: chooser}
}

type graveyardCardsToTop struct {
	cards   CardSet
	chooser PlayerRef
}

func (*graveyardCardsToTop) Text() string {
	return "put selected graveyard cards on top of their owners' libraries"
}
func (*graveyardCardsToTop) Properties() EffectProperties { return EffectProperties{} }
func (e *graveyardCardsToTop) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{{"cards", e.cards}, {"chooser", e.chooser}}}
}
func (e *graveyardCardsToTop) Apply(ctx *EffectContext) error {
	g := ctx.Game
	id, ok := e.chooser.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	chooser := g.GetPlayer(id)
	if chooser == nil {
		return nil
	}
	var proposed []*CardZoneMoveAction
	seen := make(map[uuid.UUID]bool)
	for _, card := range e.cards.resolveCards(ctx) {
		if seen[card.ID()] {
			continue
		}
		seen[card.ID()] = true
		visit, _ := g.cardMovementVisit(card.ID(), core.ZoneGraveyard)
		if visit != uuid.Nil {
			proposed = append(proposed, &CardZoneMoveAction{source: ctx.SourceID, card: card.ID(), visit: visit, from: core.ZoneGraveyard, destination: core.ZoneLibrary, owner: card.Owner()})
		}
	}
	var moves []*CardZoneMoveAction
	for _, player := range []Player{g.ActivePlayerObj(), g.NonActivePlayerObj()} {
		for _, proposal := range proposed {
			if proposal.owner != player.PlayerID() || g.graveyardVisit(proposal.card) != proposal.visit {
				continue
			}
			move, ok := g.effects.ApplyReplacements(proposal, g).(*CardZoneMoveAction)
			if ok && move != nil {
				moves = append(moves, move)
			}
		}
	}

	var cards []Card
	for _, move := range moves {
		if move.destination == core.ZoneLibrary && g.graveyardVisit(move.card) == move.visit {
			cards = append(cards, g.FindCardAnywhere(move.card))
		}
	}
	if len(cards) > 1 {
		var order []uuid.UUID
		if player, ok := chooser.(LibraryTopOrderChooser); ok {
			order = player.ChooseLibraryTopOrder(slices.Clone(cards), g)
		} else {
			bottom, top := chooser.ChooseScryPlacement(slices.Clone(cards), "put on top in any order", g)
			order = append(slices.Clone(bottom), top...)
		}
		if validCardIDOrder(order, cards) {
			ranks := make(map[uuid.UUID]int, len(order))
			for i, id := range order {
				ranks[id] = i
			}
			slices.SortStableFunc(moves, func(a, b *CardZoneMoveAction) int { return ranks[a.card] - ranks[b.card] })
		}
	}
	var events []core.GameEvent
	counts := make(map[uuid.UUID]int)
	for _, move := range slices.Backward(moves) {
		moved := g.performCardZoneMoveEvents(move)
		for _, event := range moved {
			if event.Type == core.EvtCardsLeftGraveyard {
				counts[event.PlayerID] += event.Amount
			} else {
				events = append(events, event)
			}
		}
	}
	for _, event := range events {
		g.FireEvent(event)
	}
	for _, player := range g.players {
		if count := counts[player.PlayerID()]; count > 0 {
			g.FireEvent(core.GameEvent{Type: core.EvtCardsLeftGraveyard, PlayerID: player.PlayerID(), Amount: count})
		}
	}
	return nil
}

func validCardIDOrder(order []uuid.UUID, cards []Card) bool {
	if len(order) != len(cards) {
		return false
	}
	remaining := make(map[uuid.UUID]bool, len(cards))
	for _, card := range cards {
		remaining[card.ID()] = true
	}
	for _, id := range order {
		if !remaining[id] {
			return false
		}
		delete(remaining, id)
	}
	return true
}
