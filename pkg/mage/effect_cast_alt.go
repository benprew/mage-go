package mage

import (
	"fmt"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
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
		for _, ec := range g.exile {
			if ec.Card.ID() == cardID && ec.Card.Owner() == playerID {
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
		c, ok := p.RemoveFromGraveyard(cardID)
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
	return g.castCardFromZone(playerID, cardID, zone, targets, xValue, nil)
}

// CastCardFromZoneWithAlternateCost casts the named card from the given zone
// by paying the supplied alternate mana cost instead of its printed mana cost
// (CR 117.9). Additional costs on the card are still paid afterward. The
// alternate cost is paid from the player's mana pool; the caller is
// responsible for ensuring the pool holds enough mana before calling.
func (g *Game) CastCardFromZoneWithAlternateCost(playerID, cardID uuid.UUID, zone Zone, alternate ManaCost, targets []uuid.UUID, xValue int) error {
	mc := alternate
	return g.castCardFromZone(playerID, cardID, zone, targets, xValue, &mc)
}

// castCardFromZone is the shared implementation for the public cast-from-zone
// helpers. If alternateMC is nil, no mana cost is paid (free cast). If
// alternateMC is non-nil, the alternate mana cost is paid from the pool.
func (g *Game) castCardFromZone(playerID, cardID uuid.UUID, zone Zone, targets []uuid.UUID, xValue int, alternateMC *ManaCost) error {
	p := g.GetPlayer(playerID)
	if p == nil {
		return ErrPlayerNotFound
	}
	card := g.findCardInZone(playerID, cardID, zone)
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

	// Pay the alternate mana cost if specified.
	if alternateMC != nil && !alternateMC.IsZero() {
		if !p.ManaPool().CanPay(*alternateMC) {
			return fmt.Errorf("cannot pay alternate cost %s for %s", alternateMC, card.Name())
		}
		if err := p.ManaPool().Pay(*alternateMC); err != nil {
			return err
		}
	}

	// Pay additional costs (sacrifice, discard, exile cards, etc.) printed
	// on the card. Per CR 601.2b alternate costs do NOT replace additional
	// costs; both are paid.
	if bc, ok := card.(*BaseCard); ok {
		for _, cost := range bc.AdditionalCosts() {
			if !cost.CanPay(card.ID(), playerID, g) {
				return fmt.Errorf("cannot pay additional cost for %s: %s", card.Name(), cost.Text())
			}
			if err := cost.Pay(card.ID(), playerID, g); err != nil {
				return err
			}
		}
	}

	if g.currentX != 0 && xValue == 0 {
		xValue = g.currentX
		g.currentX = 0
	}

	// Remove the card from its source zone.
	if g.removeCardFromZone(playerID, cardID, zone) == nil {
		return fmt.Errorf("could not remove %s from %s", card.Name(), zone)
	}

	// Build the spell ability effect list. Modal spells route through
	// gatherModalSpellTargets so the chosen mode supplies its own
	// targets and effects (CR 700.2).
	var effects []Effect
	var modalTargets [][]uuid.UUID
	modeChoice := 0
	if ms, ok := getModalSpellAbility(card); ok {
		mIdx, mTargets := g.gatherModalSpellTargets(p, card, ms)
		modeChoice = mIdx
		effects = append(effects, ms.modes[mIdx].Effects...)
		targets = mTargets
		modalTargets = make([][]uuid.UUID, len(ms.modes))
		modalTargets[mIdx] = mTargets
	} else {
		for _, a := range card.Abilities() {
			if sa, ok := a.(*SpellAbility); ok {
				effects = append(effects, sa.Effects()...)
			}
		}
	}

	obj := &StackObject{
		ID:           uuid.New(),
		Card:         card,
		Controller:   playerID,
		SourceID:     card.ID(),
		Effects:      effects,
		Targets:      targets,
		XValue:       xValue,
		ModeChoice:   modeChoice,
		ModalTargets: modalTargets,
	}

	if modes := card.Modes(); len(modes) > 0 {
		obj.ModeChoice = p.ChooseMode(modes, card.Name())
	}

	for _, eff := range effects {
		if !IsDividedDamageEffect(eff) {
			continue
		}
		total := DividedDamageTotal(eff).Resolve(g, card.ID(), playerID, targets)
		if total > 0 && len(targets) > 0 {
			dist := p.ChooseDamageDistribution(targets, total, card.Name(), g)
			obj.DamageDistribution = sanitizeDamageDistribution(dist, targets, total)
		}
		break
	}

	g.stack.Push(obj)

	if card.HasType(TypeInstant) {
		g.instantsCastThisTurn[playerID]++
	}

	g.FireEvent(GameEvent{
		Type:     EvtSpellCast,
		SourceID: card.ID(),
		PlayerID: playerID,
	})

	g.fireBecomesTargetEvents(obj, false)

	return nil
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
	g.castFromExilePermissions = append(g.castFromExilePermissions, CastableFromExilePermission{
		CardID:       cardID,
		PlayerID:     playerID,
		AnyColorMana: anyColorMana,
	})
}

// CastFromExilePermissionFor returns the permission record (if any) granting
// the given player the right to cast the given exiled card.
func (g *Game) CastFromExilePermissionFor(playerID, cardID uuid.UUID) *CastableFromExilePermission {
	for i := range g.castFromExilePermissions {
		perm := &g.castFromExilePermissions[i]
		if perm.PlayerID == playerID && perm.CardID == cardID {
			return perm
		}
	}
	return nil
}

// ClearCastFromExilePermission removes any permission tied to the given card
// (called when the card leaves exile).
func (g *Game) ClearCastFromExilePermission(cardID uuid.UUID) {
	kept := g.castFromExilePermissions[:0]
	for _, perm := range g.castFromExilePermissions {
		if perm.CardID != cardID {
			kept = append(kept, perm)
		}
	}
	g.castFromExilePermissions = kept
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
	card := g.findCardInZone(playerID, cardID, ZoneExile)
	if card == nil {
		return fmt.Errorf("card not in exile")
	}

	mc := card.ManaCost()
	if mc.HasX {
		mc.Generic += xValue * mc.XCount
	}

	if perm.AnyColorMana {
		// CR 609.4b: spend any-color mana for colored pips. Implement by
		// summing colored requirements into generic and using only the
		// generic-paying path. Order: pay colored first if pool has them,
		// then collapse remainder to generic.
		totalColored := mc.White + mc.Blue + mc.Black + mc.Red + mc.Green + len(mc.Hybrid)
		flat := ManaCost{Generic: mc.Generic + totalColored}
		p := g.GetPlayer(playerID)
		if !p.ManaPool().CanPay(flat) {
			return fmt.Errorf("cannot pay %s for %s", flat, card.Name())
		}
		if err := p.ManaPool().Pay(flat); err != nil {
			return err
		}
	} else {
		p := g.GetPlayer(playerID)
		if !mc.IsZero() {
			if !p.ManaPool().CanPay(mc) {
				return fmt.Errorf("cannot pay %s for %s", mc, card.Name())
			}
			if err := p.ManaPool().Pay(mc); err != nil {
				return err
			}
		}
	}

	if bc, ok := card.(*BaseCard); ok {
		for _, cost := range bc.AdditionalCosts() {
			if !cost.CanPay(card.ID(), playerID, g) {
				return fmt.Errorf("cannot pay additional cost for %s: %s", card.Name(), cost.Text())
			}
			if err := cost.Pay(card.ID(), playerID, g); err != nil {
				return err
			}
		}
	}

	g.ClearCastFromExilePermission(cardID)
	if g.removeCardFromZone(playerID, cardID, ZoneExile) == nil {
		return fmt.Errorf("could not remove %s from exile", card.Name())
	}

	pl := g.GetPlayer(playerID)
	var effects []Effect
	var modalTargets [][]uuid.UUID
	modeChoice := 0
	if ms, ok := getModalSpellAbility(card); ok {
		mIdx, mTargets := g.gatherModalSpellTargets(pl, card, ms)
		modeChoice = mIdx
		effects = append(effects, ms.modes[mIdx].Effects...)
		targets = mTargets
		modalTargets = make([][]uuid.UUID, len(ms.modes))
		modalTargets[mIdx] = mTargets
	} else {
		for _, a := range card.Abilities() {
			if sa, ok := a.(*SpellAbility); ok {
				effects = append(effects, sa.Effects()...)
			}
		}
	}
	obj := &StackObject{
		ID:           uuid.New(),
		Card:         card,
		Controller:   playerID,
		SourceID:     card.ID(),
		Effects:      effects,
		Targets:      targets,
		XValue:       xValue,
		ModeChoice:   modeChoice,
		ModalTargets: modalTargets,
	}
	if modes := card.Modes(); len(modes) > 0 {
		obj.ModeChoice = pl.ChooseMode(modes, card.Name())
	}
	g.stack.Push(obj)
	if card.HasType(TypeInstant) {
		g.instantsCastThisTurn[playerID]++
	}
	g.FireEvent(GameEvent{
		Type:     EvtSpellCast,
		SourceID: card.ID(),
		PlayerID: playerID,
	})
	g.fireBecomesTargetEvents(obj, false)
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
	switch act := a.(type) {
	case *DestroyPermanentAction:
		perm := g.FindPermanent(act.PermanentID())
		if perm == nil {
			return false
		}
		return perm.Card != nil && perm.Card.ID() == r.cardID
	}
	return false
}

func (r *exileInsteadOfGraveyardReplacement) Replace(a Action, g *Game) Action {
	switch act := a.(type) {
	case *DestroyPermanentAction:
		perm := g.FindPermanent(act.PermanentID())
		if perm == nil {
			return nil
		}
		card := perm.Card
		g.RemoveFromBattlefield(perm)
		g.ExileCard(card, r.sourceID)
		return nil
	}
	return a
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
	g.exileInsteadCards[cardID] = sourceID
}

// IsCardMarkedExileInsteadOfGraveyard returns true when the card with the
// given ID has been tagged for the "if would be put into a graveyard, exile
// it instead" replacement this turn. ResolveStackObject reads this for
// instants and sorceries that resolve and would normally go to the graveyard.
func (g *Game) IsCardMarkedExileInsteadOfGraveyard(cardID uuid.UUID) bool {
	_, ok := g.exileInsteadCards[cardID]
	return ok
}

// ClearExileInsteadOfGraveyardForTurn clears the per-turn map. Called from
// the cleanup phase together with replacement EOT cleanup.
func (g *Game) ClearExileInsteadOfGraveyardForTurn() {
	g.exileInsteadCards = map[uuid.UUID]uuid.UUID{}
}
