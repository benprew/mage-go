package mage

import (
	. "github.com/mage/mage/pkg/mage/core"
	"github.com/google/uuid"
)

// Combat manages the combat phase.
type Combat struct {
	Groups      []*CombatGroup
	Attackers   map[uuid.UUID]bool
	FirstStruck map[uuid.UUID]bool
	Bands       map[uuid.UUID][]uuid.UUID // band leader -> members (all creatures in the band)
}

// CombatGroup represents an attacker and its blockers.
type CombatGroup struct {
	AttackerID uuid.UUID
	BlockerIDs []uuid.UUID
	DefenderID uuid.UUID
}

func NewCombat() *Combat {
	return &Combat{
		Attackers:   make(map[uuid.UUID]bool),
		FirstStruck: make(map[uuid.UUID]bool),
		Bands:       make(map[uuid.UUID][]uuid.UUID),
	}
}

func (c *Combat) Reset() {
	c.Groups = nil
	c.Attackers = make(map[uuid.UUID]bool)
	c.FirstStruck = make(map[uuid.UUID]bool)
	c.Bands = make(map[uuid.UUID][]uuid.UUID)
}

func (c *Combat) AddAttacker(attackerID, defenderID uuid.UUID) {
	c.Attackers[attackerID] = true
	c.Groups = append(c.Groups, &CombatGroup{
		AttackerID: attackerID,
		DefenderID: defenderID,
	})
}

func (c *Combat) AddBlocker(blockerID, attackerID uuid.UUID) {
	for _, g := range c.Groups {
		if g.AttackerID == attackerID {
			g.BlockerIDs = append(g.BlockerIDs, blockerID)
			return
		}
	}
}

func (c *Combat) IsAttacking(id uuid.UUID) bool {
	return c.Attackers[id]
}

// AddBand records a group of creatures attacking as a band.
func (c *Combat) AddBand(members []uuid.UUID) {
	for _, id := range members {
		c.Bands[id] = members
	}
}

// IsInBand returns true if the creature with the given ID is part of an attacking band.
func (c *Combat) IsInBand(id uuid.UUID) bool {
	members, ok := c.Bands[id]
	return ok && len(members) > 0
}

// IsBandedWith returns true if id1 and id2 are in the same attacking band.
func (c *Combat) IsBandedWith(id1, id2 uuid.UUID) bool {
	members, ok := c.Bands[id1]
	if !ok {
		return false
	}
	for _, m := range members {
		if m == id2 {
			return true
		}
	}
	return false
}

// RemoveFromCombat removes a permanent from combat (attacker or blocker).
func (c *Combat) RemoveFromCombat(id uuid.UUID) {
	delete(c.Attackers, id)
	for _, g := range c.Groups {
		if g.AttackerID == id {
			g.AttackerID = uuid.Nil
		}
		filtered := g.BlockerIDs[:0]
		for _, bid := range g.BlockerIDs {
			if bid != id {
				filtered = append(filtered, bid)
			}
		}
		g.BlockerIDs = filtered
	}
}

func (c *Combat) GroupFor(attackerID uuid.UUID) *CombatGroup {
	for _, g := range c.Groups {
		if g.AttackerID == attackerID {
			return g
		}
	}
	return nil
}

// HasFirstStrikers returns true if any attacker or blocker has first strike or double strike.
func (c *Combat) HasFirstStrikers(g *Game) bool {
	for _, group := range c.Groups {
		atk := g.FindPermanent(group.AttackerID)
		if atk != nil && (atk.HasKeyword(FirstStrike) || atk.HasKeyword(DoubleStrike)) {
			return true
		}
		for _, bid := range group.BlockerIDs {
			blk := g.FindPermanent(bid)
			if blk != nil && (blk.HasKeyword(FirstStrike) || blk.HasKeyword(DoubleStrike)) {
				return true
			}
		}
	}
	return false
}

// DealsDamageInStep returns whether a permanent deals damage in the given step.
func (c *Combat) DealsDamageInStep(p *Permanent, isFirstStrikeStep bool) bool {
	hasFS := p.HasKeyword(FirstStrike)
	hasDS := p.HasKeyword(DoubleStrike)

	if isFirstStrikeStep {
		if hasFS || hasDS {
			c.FirstStruck[p.ID()] = true
			return true
		}
		return false
	}
	// Normal damage step
	if hasDS {
		return true // double strike deals damage again
	}
	return !c.FirstStruck[p.ID()] // only if didn't already deal first-strike damage
}

// CanBlock returns true if blocker can legally block the attacker.
func CanBlock(blocker, attacker *Permanent, g *Game) bool {
	// Unblockable creatures can't be blocked
	if attacker.HasKeyword(UnblockableKW) {
		return false
	}
	// CantBeBlockedByWalls creatures can't be blocked by Walls
	if attacker.HasKeyword(CantBeBlockedByWalls) && blocker.HasSubType("Wall") {
		return false
	}
	// Defender creatures can't attack (checked elsewhere), but they CAN block.
	// Protection: creature with protection from X can't be blocked by X
	if attacker.HasProtectionFrom(blocker.Card) {
		return false
	}
	// Flying: can only be blocked by creatures with flying or reach
	if attacker.HasKeyword(Flying) {
		if !blocker.HasKeyword(Flying) && !blocker.HasKeyword(Reach) {
			return false
		}
	}
	// Fear: can only be blocked by artifact creatures or black creatures
	if attacker.HasKeyword(Fear) {
		isArtifact := blocker.HasType(TypeArtifact)
		isBlack := false
		for _, col := range blocker.Card.ManaCost().Colors() {
			if col == Black {
				isBlack = true
				break
			}
		}
		if !isArtifact && !isBlack {
			return false
		}
	}
	// CantBeBlockedExceptByWalls: can only be blocked by Walls (e.g. Invisibility)
	if attacker.HasKeyword(CantBeBlockedExceptByWalls) && !blocker.HasSubType("Wall") {
		return false
	}
	// Menace: must be blocked by two or more creatures (simplified - we don't enforce here)
	return true
}

// CanAttackCheck returns true if a creature is allowed to attack (checks Defender, etc.).
func CanAttackCheck(perm *Permanent, g *Game) bool {
	return !perm.HasKeyword(Defender)
}

// HasLandwalkEvasion returns true if the attacker has a landwalk ability
// and the defending player controls a land of the matching subtype.
func HasLandwalkEvasion(attacker *Permanent, defenderID uuid.UUID, g *Game) bool {
	for _, kw := range []Keyword{Forestwalk, Islandwalk, Swampwalk, Mountainwalk, Plainswalk} {
		if attacker.HasKeyword(kw) {
			subtype := kw.LandwalkSubtype()
			for _, p := range g.Battlefield {
				if p.Controller == defenderID && p.HasType(TypeLand) && p.HasSubType(subtype) {
					return true
				}
			}
		}
	}
	return false
}
