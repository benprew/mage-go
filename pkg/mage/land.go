package mage

import (
	"fmt"
	"math"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// MaxLandPlays returns the maximum number of lands that can be played this turn.
func (g *Game) MaxLandPlays() int {
	activeID := g.ActivePlayerObj().PlayerID()
	if g.HasUnlimitedLandPlays(activeID) {
		return math.MaxInt
	}
	limit := 1
	limit += g.trackers.Turn.ExtraLandPlays(activeID)
	limit += g.effects.Rules.AdditionalLandPlays(activeID)
	return limit
}

// GrantExtraLandPlay increases the active player's land-play allowance by n
// for the current turn. Cumulative across multiple grants. Reset alongside
// landsPlayedThisTurn during the cleanup step. Implements the rules-modifier
// half of Explore / Walking Atlas / Azusa-style effects.
func (g *Game) GrantExtraLandPlay(playerID uuid.UUID, n int) {
	if n <= 0 {
		return
	}
	g.trackers.Turn.AddExtraLandPlays(playerID, n)
}

// ExtraLandPlaysGrantedThisTurn returns the cumulative additional land-play
// allowance granted to the player this turn (not including the base 1).
func (g *Game) ExtraLandPlaysGrantedThisTurn(playerID uuid.UUID) int {
	return g.trackers.Turn.ExtraLandPlays(playerID)
}

// AddRevealedTopCardEffect marks playerID as playing with the top card of
// their library revealed for this Apply() cycle (Future Sight, Oracle of
// Mul Daya, Magus of the Future). Card implementations should normally
// register the RevealTopCardOfLibrary continuous effect via
// WithStaticAbility; this method is the direct entry point for callers
// (UI, AI) and tests that need to set the flag explicitly.
func (g *Game) AddRevealedTopCardEffect(playerID uuid.UUID) {
	g.effects.Rules.AddRevealedTopCard(playerID)
}

// IsTopCardRevealed reports whether the given player is currently playing
// with the top card of their library revealed.
func (g *Game) IsTopCardRevealed(playerID uuid.UUID) bool {
	return g.effects.Rules.IsTopCardRevealed(playerID)
}

// AddPlayLandsFromZone permits playerID to play lands from the given zone
// (in addition to their hand) for this Apply() cycle. Used by Oracle of
// Mul Daya and similar cards via PlayLandsFromTopOfLibrary.
func (g *Game) AddPlayLandsFromZone(playerID uuid.UUID, zone Zone) {
	g.effects.Rules.AddPlayLandsFromZone(playerID, zone)
}

// CanPlayLandsFromZone reports whether playerID may currently play lands
// from the given zone.
func (g *Game) CanPlayLandsFromZone(playerID uuid.UUID, zone Zone) bool {
	return g.effects.Rules.CanPlayLandsFromZone(playerID, zone)
}

// AddAdditionalLandPlay registers a static-ability per-cycle additional
// land-play allowance for the player. Re-registered each Apply() cycle by
// the source's continuous effect, so it auto-clears when the source
// leaves the battlefield (Azusa, Oracle of Mul Daya, Exploration).
func (g *Game) AddAdditionalLandPlay(playerID uuid.UUID, n int) {
	g.effects.Rules.AddAdditionalLandPlay(playerID, n)
}

// playLandCore moves a land from a player's hand (or, with an active
// AddPlayLandsFromZone(ZoneLibrary) grant, from the top of their library)
// to the battlefield and fires landfall triggers, but does NOT resolve the
// stack. Callers are responsible for draining the stack (via ResolveStack
// or RunPriorityRound).
func (g *Game) playLandCore(playerID, cardID uuid.UUID) error {
	if g.StackSize() != 0 {
		return fmt.Errorf("can only play lands when the stack is empty")
	}
	if g.effects.Rules.CantPlayLands() {
		return fmt.Errorf("players can't play lands")
	}
	if !g.turns.Step().IsMainPhase() {
		return fmt.Errorf("can only play lands during a main phase")
	}
	if g.ActivePlayerObj().PlayerID() != playerID {
		return fmt.Errorf("only the active player can play a land")
	}
	if g.trackers.Turn.LandsPlayed() >= g.MaxLandPlays() {
		return fmt.Errorf("already played a land this turn")
	}

	p := g.GetPlayer(playerID)
	if p == nil {
		return ErrPlayerNotFound
	}

	var card Card
	fromZone := ZoneHand
	for _, candidate := range p.Hand() {
		if candidate.ID() == cardID {
			card = candidate
			break
		}
	}
	if card == nil && g.effects.Rules.CanPlayLandsFromZone(playerID, ZoneLibrary) {
		library := p.Library()
		if len(library) > 0 && library[0].ID() == cardID {
			card = library[0]
			fromZone = ZoneLibrary
		}
	}
	if card == nil {
		return ErrCardNotInHand
	}
	if !card.HasType(TypeLand) {
		return fmt.Errorf("card is not a land")
	}
	if fromZone == ZoneHand && g.isHandVisitPlayLocked(playerID, card.ID()) {
		return fmt.Errorf("can't play %s: hand visit is restricted", card.Name())
	}
	if g.IsLandPlayProhibited(card) {
		return fmt.Errorf("can't play %s: land play is prohibited", card.Name())
	}
	if fromZone == ZoneLibrary {
		p.SetLibrary(p.Library()[1:])
	} else {
		p.RemoveFromHand(cardID)
	}

	land := g.PutOnBattlefieldBy(card, playerID, playerID)
	g.trackers.Turn.RecordLandPlayed()

	var visit uuid.UUID
	if land != nil {
		visit = land.incarnationID
	}
	g.FireEvent(GameEvent{
		Type:              EvtLandPlayed,
		SourceID:          card.ID(),
		SourceIncarnation: visit,
		PlayerID:          playerID,
		Amount:            g.trackers.Turn.LandsPlayed(),
	})
	g.PutTriggersOnStack()

	return nil
}

// PlayLand moves a land from a player's hand to the battlefield.
func (g *Game) PlayLand(playerID, cardID uuid.UUID) error {
	if err := g.playLandCore(playerID, cardID); err != nil {
		return err
	}
	g.ResolveStack()
	return nil
}
