package mage

import "github.com/google/uuid"

// Combat manages the combat phase.
type Combat struct {
	Groups      []*CombatGroup
	Attackers   map[uuid.UUID]bool
	FirstStruck map[uuid.UUID]bool
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
	}
}

func (c *Combat) Reset() {
	c.Groups = nil
	c.Attackers = make(map[uuid.UUID]bool)
	c.FirstStruck = make(map[uuid.UUID]bool)
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
		if atk != nil && (atk.HasAbility(FirstStrike) || atk.HasAbility(DoubleStrike)) {
			return true
		}
		for _, bid := range group.BlockerIDs {
			blk := g.FindPermanent(bid)
			if blk != nil && (blk.HasAbility(FirstStrike) || blk.HasAbility(DoubleStrike)) {
				return true
			}
		}
	}
	return false
}

// DealsDamageInStep returns whether a permanent deals damage in the given step.
func (c *Combat) DealsDamageInStep(p *Permanent, isFirstStrikeStep bool) bool {
	hasFS := p.HasAbility(FirstStrike)
	hasDS := p.HasAbility(DoubleStrike)

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
	// Defender creatures can't attack (checked elsewhere), but they CAN block.
	// Protection: creature with protection from X can't be blocked by X
	if attacker.HasProtectionFrom(blocker.Card) {
		return false
	}
	// Flying: can only be blocked by creatures with flying or reach
	if attacker.HasAbility(Flying) {
		if !blocker.HasAbility(Flying) && !blocker.HasAbility(Reach) {
			return false
		}
	}
	// Fear: can only be blocked by artifact creatures or black creatures
	if attacker.HasAbility(Fear) {
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
	// Menace: must be blocked by two or more creatures (simplified - we don't enforce here)
	return true
}

// CanAttackCheck returns true if a creature is allowed to attack (checks Defender, etc.).
func CanAttackCheck(perm *Permanent, g *Game) bool {
	if perm.HasAbility(Defender) {
		return false
	}
	return true
}

// HasLandwalkEvasion returns true if the attacker has a landwalk ability
// and the defending player controls a land of the matching subtype.
func HasLandwalkEvasion(attacker *Permanent, defenderID uuid.UUID, g *Game) bool {
	for _, kw := range []Keyword{Forestwalk, Islandwalk, Swampwalk, Mountainwalk, Plainswalk} {
		if attacker.HasAbility(kw) {
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
