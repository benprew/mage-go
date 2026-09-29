package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ExilePermanent removes a permanent from the battlefield to exile and fires
// an EvtZoneChange{From: Battlefield, To: Exile}. Self-referencing leave
// triggers fire via the LKI snapshot's captured abilities per CR 603.6c.
func (g *Game) ExilePermanent(perm *Permanent) {
	controller := perm.ControllerID()
	permID := perm.ID()
	card := perm.Card
	isToken := perm.IsToken
	g.RemoveFromBattlefield(perm)
	selfAbilities := g.LKIAbilities(permID)
	if !isToken {
		g.zones.ExileCard(card, uuid.Nil)
	}
	zoneEvt := GameEvent{
		Type:     EvtZoneChange,
		SourceID: permID,
		PlayerID: controller,
		FromZone: ZoneBattlefield,
		ToZone:   ZoneExile,
	}
	g.FireEvent(zoneEvt)
	g.checkAbilitiesForEvent(selfAbilities, &zoneEvt, permID, controller)
	g.consumePendingExileZoneChange(permID)
}

// ExileCard moves a card (from any zone) to the exile zone.
func (g *Game) ExileCard(card Card, exiledBy uuid.UUID) {
	g.zones.ExileCard(card, exiledBy)
	g.recordCardPutIntoExile(card)
}

// ExileCardFaceDown moves a card to the exile zone face down. Only players in
// revealedTo may inspect the card's identity (via FindExiledCard / GetExile +
// ExiledCard.VisibleTo). For Gonti-style "exile face down" effects, only the
// exiling player (not the card's owner) sees the identity — pass that player's
// ID in revealedTo. Owners of face-down exiled cards do not automatically see
// the identity, matching Gonti's printed ruling.
func (g *Game) ExileCardFaceDown(card Card, exiledBy uuid.UUID, revealedTo ...uuid.UUID) {
	g.zones.ExileCardFaceDown(card, exiledBy, revealedTo...)
	g.recordCardPutIntoExile(card)
}

// RevealExiledCardTo grants the given player permission to see the identity of
// the face-down exiled card with the given ID. No-op if the card is face up
// (already public) or not in exile.
func (g *Game) RevealExiledCardTo(cardID, playerID uuid.UUID) {
	g.zones.RevealExiledCardTo(cardID, playerID)
}

// FindExiledCard finds an exiled card by its ID.
func (g *Game) FindExiledCard(cardID uuid.UUID) *ExiledCard {
	return g.zones.FindExiledCard(cardID)
}

// RemoveFromExile removes a card from exile by ID and returns it.
func (g *Game) RemoveFromExile(cardID uuid.UUID) (Card, bool) {
	return g.zones.RemoveFromExile(cardID)
}

// RemoveExiledCardBySource removes all exiled cards with the given ExiledBy ID
// and returns them. Used by Tawnos's Coffin and similar cards.
func (g *Game) RemoveExiledCardBySource(exiledBy uuid.UUID) []ExiledCard {
	return g.zones.RemoveExiledCardBySource(exiledBy)
}
