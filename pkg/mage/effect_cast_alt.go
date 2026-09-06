package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// Cast-from-non-hand-zone helpers.
//
// CR 117.9 / 601.2b: a player may sometimes be allowed to cast a card from a
// zone other than their hand, sometimes "without paying its mana cost" (an
// alternate cost of 0 mana per CR 118.9), and sometimes by paying a different
// alternate cost. These helpers package the tail of the cast pipeline (zone
// removal, stack-object construction, ModeChoice / divided-damage prompts,
// EvtSpellCast event) so the engine and individual cards can grant such
// permissions without re-implementing CastSpellByName.
//
// The helpers do NOT validate sorcery-speed timing: the granting effect
// decides when the permission applies. They DO validate card location and
// fire EvtSpellCast so cast-triggered abilities (Storm, prowess, etc.) see
// the cast.

// findCardInZone returns the Card with the given ID in the given player's
// zone, or nil if not present. Stack and battlefield are not supported here
// (those are not cast-from zones in the normal sense).
func (g *Game) findCardInZone(playerID, cardID uuid.UUID, zone Zone) Card {
	return g.findCardInZoneOpt(playerID, cardID, zone, false)
}

// findCardInZoneOpt is like findCardInZone but, when permitForeignOwner is
// true, does not require an exiled card to be owned by playerID. This
// supports effects (e.g. Etali, Primal Storm) that exile cards from each
// player's library and let one player cast any of them, regardless of
// original ownership.
func (g *Game) findCardInZoneOpt(playerID, cardID uuid.UUID, zone Zone, permitForeignOwner bool) Card {
	p := g.GetPlayer(playerID)
	if p == nil {
		return nil
	}
	switch zone {
	case ZoneHand:
		for _, c := range p.Hand() {
			if c.ID() == cardID {
				return c
			}
		}
	case ZoneGraveyard:
		for _, c := range p.Graveyard() {
			if c.ID() == cardID {
				return c
			}
		}
	case ZoneLibrary:
		for _, c := range p.Library() {
			if c.ID() == cardID {
				return c
			}
		}
	case ZoneExile:
		for _, ec := range g.zones.exile {
			if ec.Card.ID() != cardID {
				continue
			}
			if permitForeignOwner || ec.Card.Owner() == playerID {
				return ec.Card
			}
		}
	}
	return nil
}

// removeCardFromZone removes the card with the given ID from the given
// player's zone. Returns the removed card, or nil if not present.
func (g *Game) removeCardFromZone(playerID, cardID uuid.UUID, zone Zone) Card {
	p := g.GetPlayer(playerID)
	if p == nil {
		return nil
	}
	switch zone {
	case ZoneHand:
		c, ok := p.RemoveFromHand(cardID)
		if !ok {
			return nil
		}
		return c
	case ZoneGraveyard:
		c, ok := g.MoveFromGraveyard(playerID, cardID, ZoneStack)
		if !ok {
			return nil
		}
		return c
	case ZoneLibrary:
		lib := p.Library()
		for i, c := range lib {
			if c.ID() == cardID {
				newLib := append([]Card{}, lib[:i]...)
				newLib = append(newLib, lib[i+1:]...)
				p.SetLibrary(newLib)
				return c
			}
		}
	case ZoneExile:
		c, ok := g.RemoveFromExile(cardID)
		if !ok {
			return nil
		}
		return c
	}
	return nil
}

// CastCardFromZoneWithoutPaying puts the named card on the stack from the
// given zone, paying no mana cost. Additional costs printed on the card
// (sacrifice, discard, etc.) are still paid (CR 601.2b, 118.9).
//
// X is set to xValue (used for spells with X in their cost; "without paying
// its mana cost" sets X = 0 unless the alternate cost specifies otherwise —
// callers may pass 0 for the standard case).
func (g *Game) CastCardFromZoneWithoutPaying(playerID, cardID uuid.UUID, zone Zone, targets []uuid.UUID, xValue int) error {
	return g.castCardFromZone(playerID, cardID, zone, targets, xValue, nil, false)
}

// CastCardFromExileWithoutPaying is like CastCardFromZoneWithoutPaying for
// ZoneExile, but does not require the caster to be the card's owner. Per
// CR 706.10, the player casting the spell becomes its controller regardless
// of ownership; this helper supports effects (e.g. Etali, Primal Storm) that
// exile cards from each player's library and grant one player the option
// to cast any of them.
func (g *Game) CastCardFromExileWithoutPaying(playerID, cardID uuid.UUID, targets []uuid.UUID, xValue int) error {
	return g.castCardFromZone(playerID, cardID, ZoneExile, targets, xValue, nil, true)
}

// CastCardFromZoneWithAlternateCost casts the named card from the given zone
// by paying the supplied alternate mana cost instead of its printed mana cost
// (CR 117.9). Additional costs on the card are still paid afterward. The
// alternate cost and every additional/action mana component are planned and
// auto-paid as one locked total cost.
func (g *Game) CastCardFromZoneWithAlternateCost(playerID, cardID uuid.UUID, zone Zone, alternate ManaCost, targets []uuid.UUID, xValue int) error {
	mc := alternate
	return g.castCardFromZone(playerID, cardID, zone, targets, xValue, &mc, false)
}

// castCardFromZone is the shared implementation for the public cast-from-zone
// helpers. If alternateMC is nil, no mana cost is paid (free cast). If
// alternateMC is non-nil, the alternate mana cost is paid from the pool.
// If permitForeignOwner is true, an exiled card may be cast even if its
// owner is not playerID (per CR 706.10 the caster becomes controller).
func (g *Game) castCardFromZone(playerID, cardID uuid.UUID, zone Zone, targets []uuid.UUID, xValue int, alternateMC *ManaCost, permitForeignOwner bool) error {
	return g.castCardFromZoneOpts(playerID, cardID, zone, targets, xValue, alternateMC, permitForeignOwner, false)
}

// castCardFromZoneOpts is the underlying implementation; exileOnLeaveStack
// causes the resolver to send the card to exile instead of graveyard on
// resolution / fizzle (used by flashback per CR 702.34).
func (g *Game) castCardFromZoneOpts(playerID, cardID uuid.UUID, zone Zone, targets []uuid.UUID, xValue int, alternateMC *ManaCost, permitForeignOwner, exileOnLeaveStack bool) error {
	return g.castCardFromZoneOptsWithCosts(playerID, cardID, zone, targets, xValue, alternateMC, permitForeignOwner, exileOnLeaveStack, nil, false)
}

func (g *Game) castCardFromZoneOptsWithCosts(playerID, cardID uuid.UUID, zone Zone, targets []uuid.UUID, xValue int, alternateMC *ManaCost, permitForeignOwner, exileOnLeaveStack bool, extraCosts []Cost, anyColor bool) error {
	if g.GetPlayer(playerID) == nil {
		return ErrPlayerNotFound
	}
	card := g.findCardInZoneOpt(playerID, cardID, zone, permitForeignOwner)
	if card == nil {
		return fmt.Errorf("card %s not found in %s", cardID, zone)
	}

	// CR 117.6: a player can't cast a card with no mana cost using an
	// alternate cost that doesn't specify an alternative; "without paying
	// its mana cost" likewise can't be used to cast a card with no mana
	// cost. Lands aren't cast.
	if card.HasType(TypeLand) {
		return fmt.Errorf("can't cast a land")
	}

	var mana ManaCost
	if alternateMC != nil {
		mana = *alternateMC
	}
	totalCosts := append([]Cost(nil), extraCosts...)
	if bc, ok := card.(*BaseCard); ok {
		totalCosts = append(totalCosts, bc.AdditionalCosts()...)
	}
	var actionCosts []Cost
	for _, ability := range card.Abilities() {
		if spell, ok := ability.(*SpellAbility); ok && spell.Kind() == ActionSpell {
			actionCosts = append(actionCosts, spell.Costs()...)
		}
	}
	totalCosts = append(totalCosts, g.prepareActionCosts(actionCosts, targets, xValue)...)

	payment, err := g.prepareSpellPaymentTransaction(spellPaymentSpec{
		Card:       card,
		Controller: playerID,
		Zone:       zone,
		Mana:       mana,
		Costs:      totalCosts,
		Targets:    targets,
		XValue:     xValue,
		AnyColor:   anyColor,
	})
	if err != nil {
		return fmt.Errorf("cannot pay total cost for %s: %w", card.Name(), err)
	}
	if err := payment.Commit(); err != nil {
		return err
	}

	if g.resolution.X() != 0 && xValue == 0 {
		xValue = g.resolution.X()
		g.resolution.SetX(0)
	}

	_, err = g.pushCastSpellObject(castStackObjectOptions{
		Card:              card,
		Controller:        playerID,
		Targets:           targets,
		XValue:            xValue,
		CastZone:          zone,
		ExileOnLeaveStack: exileOnLeaveStack,
		SnapshotCast:      true,
	})
	return err
}

// CastableFromExilePermission grants a player permission to cast a specific
// exiled card. Used by effects like Gonti, Lord of Luxury that exile a card
// and grant the controller permission to cast it for as long as it remains
// exiled. If anyColorMana is true, the controller may also spend mana as
// though it were mana of any color to cast that card (CR 609.4b).
type CastableFromExilePermission struct {
	CardID       uuid.UUID
	PlayerID     uuid.UUID
	AnyColorMana bool
}

// GrantCastFromExile records permission for the given player to cast the
// given exiled card at sorcery speed (or instant if the card is an instant).
// The permission persists until the card leaves exile. If anyColorMana is
// true, the player may spend mana of any color when paying for the card.
func (g *Game) GrantCastFromExile(playerID, cardID uuid.UUID, anyColorMana bool) {
	g.zones.GrantCastFromExile(playerID, cardID, anyColorMana)
}

// CastFromExilePermissionFor returns the permission record (if any) granting
// the given player the right to cast the given exiled card.
func (g *Game) CastFromExilePermissionFor(playerID, cardID uuid.UUID) *CastableFromExilePermission {
	return g.zones.CastFromExilePermissionFor(playerID, cardID)
}

// ClearCastFromExilePermission removes any permission tied to the given card
// (called when the card leaves exile).
func (g *Game) ClearCastFromExilePermission(cardID uuid.UUID) {
	g.zones.ClearCastFromExilePermission(cardID)
}

// CastExiledCardWithPermission casts an exiled card the controller has been
// granted permission to cast (e.g. Gonti). If the permission carries
// AnyColorMana, the alternate cost of the card's printed mana cost is paid
// as colorless equivalent (any color may pay any colored pip).
func (g *Game) CastExiledCardWithPermission(playerID, cardID uuid.UUID, targets []uuid.UUID, xValue int) error {
	perm := g.CastFromExilePermissionFor(playerID, cardID)
	if perm == nil {
		return fmt.Errorf("no permission to cast %s from exile", cardID)
	}
	// Per CR 706.10 the caster becomes controller regardless of original
	// ownership; Gonti exiles cards owned by an opponent, so we permit
	// foreign ownership when looking up the exiled card.
	card := g.findCardInZoneOpt(playerID, cardID, ZoneExile, true)
	if card == nil {
		return fmt.Errorf("card not in exile")
	}

	mc := card.ManaCost()
	if err := g.castCardFromZoneOptsWithCosts(playerID, cardID, ZoneExile, targets, xValue, &mc, true, false, nil, perm.AnyColorMana); err != nil {
		return err
	}
	g.ClearCastFromExilePermission(cardID)
	return nil
}

// --- "If would be put into a graveyard this turn, exile it instead" ---
// Scholar of the Lost Trove and similar cards: when they grant a free cast
// from graveyard, the spell goes to exile instead of the graveyard if it
// would die this turn. This is a one-shot, card-scoped replacement effect.

// exileInsteadOfGraveyardReplacement is a turn-scoped replacement effect that
// causes a specific card (by ID) to be exiled instead of going to the
// graveyard. Used by Scholar of the Lost Trove. Tracks the affected card by
// the underlying card ID, which is stable across resolution.
type exileInsteadOfGraveyardReplacement struct {
	replacementBase
	cardID uuid.UUID
}

func (r *exileInsteadOfGraveyardReplacement) Matches(a Action, g GameReader) bool {
	act, ok := a.(*DestroyPermanentAction)
	if !ok {
		return false
	}
	perm := g.FindPermanent(act.PermanentID())
	if perm == nil {
		return false
	}
	return perm.Card != nil && perm.Card.ID() == r.cardID
}

func (r *exileInsteadOfGraveyardReplacement) Replace(a Action, g *Game) Action {
	act, ok := a.(*DestroyPermanentAction)
	if !ok {
		return a
	}
	perm := g.FindPermanent(act.PermanentID())
	if perm == nil {
		return nil
	}
	card := perm.Card
	g.RemoveFromBattlefield(perm)
	g.ExileCard(card, r.sourceID)
	return nil
}

func (r *exileInsteadOfGraveyardReplacement) IsActive(_ GameReader) bool { return true }
func (r *exileInsteadOfGraveyardReplacement) Clone() ReplacementEffect {
	return &exileInsteadOfGraveyardReplacement{
		replacementBase: replacementBase{sourceID: r.sourceID, duration: r.duration},
		cardID:          r.cardID,
	}
}

// AddExileIfWouldGoToGraveyardThisTurn registers a turn-scoped replacement so
// that if the card with the given ID would be put into a graveyard from the
// battlefield this turn, it is exiled instead. Used by Scholar of the Lost
// Trove for the spell it grants a free cast of.
//
// Note: this only intercepts the destroy/death path for permanents (CR 614).
// Instants and sorceries put into the graveyard after resolving are handled
// directly by the resolver (see ResolveStackObject).
func (g *Game) AddExileIfWouldGoToGraveyardThisTurn(cardID, sourceID uuid.UUID) {
	g.effects.AddReplacement(&exileInsteadOfGraveyardReplacement{
		replacementBase: replacementBase{sourceID: sourceID, duration: EndOfTurn},
		cardID:          cardID,
	})
	g.zones.AddExileInstead(cardID, sourceID)
}

// IsCardMarkedExileInsteadOfGraveyard returns true when the card with the
// given ID has been tagged for the "if would be put into a graveyard, exile
// it instead" replacement this turn. ResolveStackObject reads this for
// instants and sorceries that resolve and would normally go to the graveyard.
func (g *Game) IsCardMarkedExileInsteadOfGraveyard(cardID uuid.UUID) bool {
	return g.zones.IsExileInstead(cardID)
}

// ClearExileInsteadOfGraveyardForTurn clears the per-turn map. Called from
// the cleanup phase together with replacement EOT cleanup.
func (g *Game) ClearExileInsteadOfGraveyardForTurn() {
	g.zones.ClearExileInstead()
}
