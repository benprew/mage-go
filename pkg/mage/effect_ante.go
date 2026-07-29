package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type cardYouOwnInAnteTarget struct {
	BaseTarget
}

// TargetCardYouOwnInAnte targets one card in the shared ante zone that the
// ability controller currently owns.
func TargetCardYouOwnInAnte() Target {
	return &cardYouOwnInAnteTarget{BaseTarget: BaseTarget{min: 1, max: 1}}
}

func (t *cardYouOwnInAnteTarget) Possible(controller uuid.UUID, _ Card, g *Game) []uuid.UUID {
	cards, err := g.AnteCardsOwnedBy(controller)
	if err != nil {
		return nil
	}
	possible := make([]uuid.UUID, 0, len(cards))
	for _, card := range cards {
		possible = append(possible, card.ID())
	}
	return possible
}

func (t *cardYouOwnInAnteTarget) Choose(controller uuid.UUID, source Card, g *Game, chosen []uuid.UUID) error {
	if len(chosen) != 1 || !slices.Contains(t.Possible(controller, source, g), chosen[0]) {
		return fmt.Errorf("must target a card you own in the ante")
	}
	t.chosen = append([]uuid.UUID(nil), chosen...)
	return nil
}

// AnteLibraryTopEffect antes the top card of selected players' libraries.
type AnteLibraryTopEffect struct {
	selector PlayerSelector
}

// AnteLibraryTop antes the top card of each selected player's library. It
// defaults to the effect controller. A card that player does not own cannot
// be anted, and one failed ante does not prevent later selected players from
// performing the action.
func AnteLibraryTop() *AnteLibraryTopEffect {
	return &AnteLibraryTopEffect{}
}

func (e *AnteLibraryTopEffect) Targeting(selector PlayerSelector) *AnteLibraryTopEffect {
	e.selector = selector
	return e
}

func (*AnteLibraryTopEffect) Text() string                 { return "ante the top card of the library" }
func (*AnteLibraryTopEffect) Properties() EffectProperties { return EffectProperties{} }

func (e *AnteLibraryTopEffect) Apply(ctx *EffectContext) error {
	selector := e.selector
	if selector == nil {
		selector = SelectController()
	}
	for _, playerID := range selector.Select(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets) {
		player := ctx.Game.GetPlayer(playerID)
		if player == nil || len(player.Library()) == 0 {
			continue
		}
		_ = ctx.Game.MoveToAnte(playerID, player.Library()[0].ID())
	}
	return nil
}

type exchangeTargetAnteCardWithLibraryTopEffect struct{}

// ExchangeTargetAnteCardWithLibraryTop exchanges the targeted card in the
// shared ante zone with the top card of the effect controller's library. The
// exchange does nothing unless the controller owns both cards and can ante
// the library card.
func ExchangeTargetAnteCardWithLibraryTop() Effect {
	return &exchangeTargetAnteCardWithLibraryTopEffect{}
}

func (*exchangeTargetAnteCardWithLibraryTopEffect) Text() string {
	return "exchange that card with the top card of your library"
}
func (*exchangeTargetAnteCardWithLibraryTopEffect) Properties() EffectProperties {
	return EffectProperties{}
}

func (*exchangeTargetAnteCardWithLibraryTopEffect) Apply(ctx *EffectContext) error {
	if len(ctx.Targets) == 0 || ctx.Targets[0] == uuid.Nil {
		return nil
	}
	player := ctx.Game.GetPlayer(ctx.Controller)
	if player == nil || len(player.Library()) == 0 {
		return nil
	}
	anteCard := ctx.Game.FindCardAnywhere(ctx.Targets[0])
	top := player.Library()[0]
	ownedAnte, err := ctx.Game.AnteCardsOwnedBy(ctx.Controller)
	if err != nil {
		return err
	}
	if anteCard == nil || !slices.ContainsFunc(ownedAnte, func(card Card) bool { return card.ID() == anteCard.ID() }) || top.Owner() != ctx.Controller || top.IsToken() {
		return nil
	}
	if err := ctx.Game.MoveToAnte(ctx.Controller, top.ID()); err != nil {
		return err
	}
	anteCard, err = ctx.Game.RemoveFromAnte(anteCard.ID(), core.ZoneLibrary)
	if err != nil {
		return err
	}
	player.SetLibrary(append([]Card{anteCard}, player.Library()...))
	return nil
}
