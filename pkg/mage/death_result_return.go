package mage

import (
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ReturnChosenGraveyardCardsForEachPermanent makes one choice for each saved visit.
// All choices precede entry. Each chosen card enters under its owner's control.
func ReturnChosenGraveyardCardsForEachPermanent(subjects *PermanentSetBinding, member *PermanentBinding, graveyard PlayerRef, filter CardFilter) ResolvingEffect {
	return &deathResultReturn{subjects, member, graveyard, filter}
}

type deathResultReturn struct {
	subjects  *PermanentSetBinding
	member    *PermanentBinding
	graveyard PlayerRef
	filter    CardFilter
}

func (*deathResultReturn) Text() string {
	return "return a chosen graveyard card for each saved permanent"
}
func (*deathResultReturn) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, Mass: true}
}
func (e *deathResultReturn) inspectDefinition() definitionNode {
	n := definitionNode{Role: "saved-permanent-return", isolatedBindings: true, children: []definitionEdge{
		{"subjects", savedPermanentSet{e.subjects}}, {"member", loopMember{e.member}}, {"graveyard", e.graveyard},
	}}
	if e.filter.IsZero() {
		n.problem = "supply a graveyard card filter"
	}
	return n
}
func (e *deathResultReturn) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.graveyard)
}
func (e *deathResultReturn) Apply(ctx *EffectContext) error {
	g := ctx.Game
	chooser := g.GetPlayer(ctx.Controller)
	if chooser == nil {
		return nil
	}
	members := slices.Clone(g.resolution.createdTokenBindings[e.subjects])
	outer := g.resolution.permanentBindings
	defer func() { g.resolution.permanentBindings = outer }()
	byOwner := make(map[uuid.UUID][]capturedCard)
	selected := make(map[uuid.UUID]bool)
	for _, member := range members {
		frame := maps.Clone(outer)
		if frame == nil {
			frame = make(map[*PermanentBinding]capturedPermanent)
		}
		frame[e.member] = member
		g.resolution.permanentBindings = frame
		id, ok := e.graveyard.resolvePlayer(ctx)
		if !ok {
			continue
		}
		player := g.GetPlayer(id)
		if player == nil {
			continue
		}
		var candidates []Card
		visits := make(map[uuid.UUID]uuid.UUID)
		for _, card := range player.Graveyard() {
			if !selected[card.ID()] && !card.IsToken() && e.filter.Match(card) {
				candidates = append(candidates, card)
				visits[card.ID()] = g.graveyardVisit(card.ID())
			}
		}
		if len(candidates) == 0 {
			continue
		}
		choice := candidates[0]
		if len(candidates) > 1 {
			chosen := chooser.ChooseCardFromGraveyard(slices.Clone(candidates), "Choose a card to return to the battlefield", g)
			for _, card := range candidates {
				if chosen != nil && card.ID() == chosen.ID() {
					choice = card
					break
				}
			}
		}
		selected[choice.ID()] = true
		byOwner[choice.Owner()] = append(byOwner[choice.Owner()], capturedCard{id: choice.ID(), visit: visits[choice.ID()], zone: core.ZoneGraveyard})
	}
	g.resolution.permanentBindings = outer
	return g.enterSavedGraveyardCards(ctx, byOwner)
}
