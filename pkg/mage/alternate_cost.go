package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// AlternateCost describes a card-level alternative casting cost (CR 117.9 /
// 118.9). A card can carry one or more AlternateCosts; each says "you may cast
// this card from <Zone> by paying <Mana> [and <Additional> ...] rather than
// paying its mana cost." Examples: Scourge of Nel Toth's "cast from your
// graveyard by paying {B}{B} and sacrificing two creatures".
//
// The alternate-cost machinery:
//   - WithAlternateCost on the card constructor records the alt-cost.
//   - CastCardWithAlternateCost on *Game pays the alt mana + additional costs
//     and routes through the standard cast-from-zone pipeline (so EvtSpellCast
//     fires, the spell stops by the stack, and the card moves to the
//     battlefield/graveyard via normal resolution).
//
// Condition is an optional predicate (e.g. "only during your turn"); when nil,
// the alt-cost is always offered as long as the card is in the named zone.
type AlternateCost struct {
	Zone       Zone
	Mana       ManaCost
	Additional []Cost
	Condition  func(g *Game, playerID uuid.UUID) bool
	// Flashback marks this alternate cost as the flashback keyword (CR 702.34).
	// When the spell is cast via this alt-cost, it is exiled instead of being
	// put into its owner's graveyard upon resolution or fizzle.
	Flashback bool
}

// WithAlternateCost registers an alternate cost on the card. The cost is paid
// instead of the card's printed mana cost when the player chooses to cast via
// this path (CR 117.9). additional costs (sacrifice, discard, pay life, etc.)
// are paid in addition to the alt mana cost.
func WithAlternateCost(zone Zone, mana ManaCost, additional ...Cost) CardOption {
	return func(c *BaseCard) {
		c.alternateCosts = append(c.alternateCosts, AlternateCost{
			Zone:       zone,
			Mana:       mana,
			Additional: additional,
		})
	}
}

// WithFlashback registers the flashback alternate cost (CR 702.34): the card
// may be cast from its owner's graveyard for the given mana cost (plus any
// additional costs). On resolution or fizzle the card is exiled instead of
// being put into the graveyard.
func WithFlashback(mana ManaCost, additional ...Cost) CardOption {
	return func(c *BaseCard) {
		c.alternateCosts = append(c.alternateCosts, AlternateCost{
			Zone:       ZoneGraveyard,
			Mana:       mana,
			Additional: additional,
			Flashback:  true,
		})
	}
}

// AlternateCosts returns the alternate-cost options registered on this card.
// Returns nil when the card has no alternate costs.
func (c *BaseCard) AlternateCosts() []AlternateCost { return c.alternateCosts }

// CastCardWithAlternateCost casts the named card via the altIdx'th alternate
// cost on the card. The card must currently be in the alt's source zone.
// The alternate mana, alternate additional costs, printed additional costs,
// and spell-action costs are locked and paid by the shared spell transaction.
func (g *Game) CastCardWithAlternateCost(playerID, cardID uuid.UUID, altIdx int, targets []uuid.UUID, xValue int) error {
	p := g.GetPlayer(playerID)
	if p == nil {
		return ErrPlayerNotFound
	}
	// Locate the card across plausible zones so we can read its
	// alternate-cost list. The alt itself names the legal source zone.
	var card Card
	for _, zone := range []Zone{ZoneGraveyard, ZoneHand, ZoneExile, ZoneLibrary} {
		if c := g.findCardInZone(playerID, cardID, zone); c != nil {
			card = c
			break
		}
	}
	if card == nil {
		return fmt.Errorf("card %s not found in any cast-from zone", cardID)
	}
	bc, ok := card.(*BaseCard)
	if !ok {
		return fmt.Errorf("card %s does not support alternate costs", card.Name())
	}
	if altIdx < 0 || altIdx >= len(bc.alternateCosts) {
		return fmt.Errorf("alternate-cost index %d out of range for %s", altIdx, card.Name())
	}
	alt := bc.alternateCosts[altIdx]

	if g.findCardInZone(playerID, cardID, alt.Zone) == nil {
		return fmt.Errorf("%s is not in %s for alt-cost", card.Name(), alt.Zone)
	}
	if alt.Condition != nil && !alt.Condition(g, playerID) {
		return fmt.Errorf("alt-cost condition not met for %s", card.Name())
	}

	mc := alt.Mana
	return g.castCardFromZoneOptsWithCosts(playerID, cardID, alt.Zone, targets, xValue, &mc, false, alt.Flashback, alt.Additional, false)
}
