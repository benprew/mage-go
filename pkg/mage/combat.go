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

// IsBlocking returns true if the creature with the given ID is a declared blocker.
func (c *Combat) IsBlocking(id uuid.UUID) bool {
	for _, g := range c.Groups {
		for _, bid := range g.BlockerIDs {
			if bid == id {
				return true
			}
		}
	}
	return false
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
	// Per-pair block restrictions from continuous effects (e.g. Argothian Pixies)
	if g.Effects.IsBlockPrevented(blocker.ID(), attacker.ID()) {
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

// HasLandwalkEvasion returns true if the attacker has a landwalk ability
// and the defending player controls a land of the matching subtype.
// Uses the data-driven landwalkSubtypes map so new landwalk variants
// (e.g. Desertwalk) work automatically.
func HasLandwalkEvasion(attacker *Permanent, defenderID uuid.UUID, g *Game) bool {
	for kw, subtype := range LandwalkAttrs() {
		if attacker.HasKeyword(kw) && !g.Effects.Rules.IsLandwalkNullified(kw) {
			for _, p := range g.Battlefield {
				if p.Controller == defenderID && p.HasType(TypeLand) && p.HasSubType(subtype) {
					return true
				}
			}
		}
	}
	// Check legendary landwalk: unblockable if defender controls a legendary land
	if attacker.HasKeyword(LegendaryLandwalk) {
		for _, p := range g.Battlefield {
			if p.Controller == defenderID && p.HasType(TypeLand) && p.Card.HasSuperType(SuperLegendary) {
				return true
			}
		}
	}
	return false
}

// ResolveDamage executes combat damage for all combat groups. It replaces the
// old Game.DoCombatDamage method and operates on the Combat receiver so the
// logic lives alongside the rest of the combat state.
func (c *Combat) ResolveDamage(g *Game, isFirstStrikeStep bool) {
	if g.Effects.Damage.PreventsCombatDamage() {
		return
	}

	// Track which band leaders we've already processed to avoid double-dealing.
	processedBands := make(map[uuid.UUID]bool)

	for _, group := range c.Groups {
		atk := g.FindPermanent(group.AttackerID)
		if atk == nil {
			continue
		}

		// Attacking band: handle all band members together under the leader.
		if c.IsInBand(group.AttackerID) {
			bandMembers := c.Bands[group.AttackerID]
			leaderID := bandMembers[0]
			if processedBands[leaderID] {
				continue
			}
			processedBands[leaderID] = true
			c.doBandedAttackDamage(g, bandMembers, group.DefenderID, isFirstStrikeStep)
			continue
		}

		// Non-banded attacker.
		if len(group.BlockerIDs) == 0 {
			// Unblocked — damage to defending player.
			if c.DealsDamageInStep(atk, isFirstStrikeStep) {
				defender := g.GetPlayer(group.DefenderID)
				if defender != nil {
					dmg := atk.CurrentPower(g)
					if dmg > 1 && g.Effects.Damage.HasForcefieldShield(group.DefenderID) {
						dmg = 1
					}
					if bgID := g.Effects.Damage.GetBodyguard(group.DefenderID); bgID != uuid.Nil {
						bg := g.FindPermanent(bgID)
						if bg != nil && !bg.Tapped {
							g.DealDamageToPermanent(bg, dmg, atk.ID())
							continue
						}
					}
					g.DealDamageToPlayer(defender, dmg, atk.ID())
				}
			}
		} else if c.isBlockingBand(g, group.BlockerIDs) {
			// Blocking band: defending player distributes attacker's damage.
			c.doBlockingBandDamage(g, atk, group, isFirstStrikeStep)
		} else {
			// Normal blocked combat.
			c.doNormalBlockedDamage(g, atk, group, isFirstStrikeStep)
		}
	}

	g.CheckStateBasedActions()
}

// isBlockingBand returns true when multiple blockers block the same attacker and
// at least one of them has banding, forming a blocking band.
func (c *Combat) isBlockingBand(g *Game, blockerIDs []uuid.UUID) bool {
	if len(blockerIDs) < 2 {
		return false
	}
	for _, bid := range blockerIDs {
		blk := g.FindPermanent(bid)
		if blk != nil && blk.HasKeyword(Banding) {
			return true
		}
	}
	return false
}

// doNormalBlockedDamage handles blocked combat for a single non-banded attacker.
func (c *Combat) doNormalBlockedDamage(g *Game, atk *Permanent, group *CombatGroup, isFirstStrikeStep bool) {
	if c.DealsDamageInStep(atk, isFirstStrikeStep) {
		remainingDmg := atk.CurrentPower(g)
		for _, bid := range group.BlockerIDs {
			blk := g.FindPermanent(bid)
			if blk == nil {
				continue
			}
			needed := blk.CurrentToughness(g) - blk.Damage
			if needed <= 0 {
				continue
			}
			dealt := min(remainingDmg, needed)
			g.DealDamageToPermanent(blk, dealt, atk.ID())
			remainingDmg -= dealt
			if remainingDmg <= 0 {
				break
			}
		}
		if remainingDmg > 0 && atk.HasKeyword(Trample) {
			defender := g.GetPlayer(group.DefenderID)
			if defender != nil {
				g.DealDamageToPlayer(defender, remainingDmg, atk.ID())
			}
		}
	}
	for _, bid := range group.BlockerIDs {
		blk := g.FindPermanent(bid)
		if blk == nil {
			continue
		}
		if c.DealsDamageInStep(blk, isFirstStrikeStep) {
			g.DealDamageToPermanent(atk, blk.CurrentPower(g), blk.ID())
		}
	}
}

// doBlockingBandDamage handles combat where multiple blockers with banding block
// a single attacker. The defending player (controller of the blocking band) chooses
// how the attacker's damage is distributed across band members.
func (c *Combat) doBlockingBandDamage(g *Game, atk *Permanent, group *CombatGroup, isFirstStrikeStep bool) {
	// Collect blocker permanents.
	var blockerPerms []*Permanent
	for _, bid := range group.BlockerIDs {
		blk := g.FindPermanent(bid)
		if blk != nil {
			blockerPerms = append(blockerPerms, blk)
		}
	}

	// Attacker deals damage — defending player distributes it across the blocking band.
	if c.DealsDamageInStep(atk, isFirstStrikeStep) {
		atkPower := atk.CurrentPower(g)

		var defendingPlayerID uuid.UUID
		if len(blockerPerms) > 0 {
			defendingPlayerID = blockerPerms[0].Controller
		}
		defendingPlayer := g.GetPlayer(defendingPlayerID)

		var distribution map[uuid.UUID]int
		if distributor, ok := defendingPlayer.(BandingDamageDistributor); ok {
			distribution = distributor.GetBandingDamageDistribution(blockerPerms)
		}

		if distribution != nil {
			usedDmg := 0
			for _, blk := range blockerPerms {
				dmg := distribution[blk.ID()]
				if dmg > 0 {
					g.DealDamageToPermanent(blk, dmg, atk.ID())
					usedDmg += dmg
				}
			}
			// Trample: any damage beyond what was distributed goes to the defending player.
			if atk.HasKeyword(Trample) {
				if trampleDmg := atkPower - usedDmg; trampleDmg > 0 {
					defender := g.GetPlayer(group.DefenderID)
					if defender != nil {
						g.DealDamageToPlayer(defender, trampleDmg, atk.ID())
					}
				}
			}
		} else {
			// Default: normal distribution (no banding benefit).
			remainingDmg := atkPower
			for _, blk := range blockerPerms {
				needed := blk.CurrentToughness(g) - blk.Damage
				if needed <= 0 {
					continue
				}
				dealt := min(remainingDmg, needed)
				g.DealDamageToPermanent(blk, dealt, atk.ID())
				remainingDmg -= dealt
				if remainingDmg <= 0 {
					break
				}
			}
			if remainingDmg > 0 && atk.HasKeyword(Trample) {
				defender := g.GetPlayer(group.DefenderID)
				if defender != nil {
					g.DealDamageToPlayer(defender, remainingDmg, atk.ID())
				}
			}
		}
	}

	// Each blocker still deals its own damage to the attacker.
	for _, blk := range blockerPerms {
		if c.DealsDamageInStep(blk, isFirstStrikeStep) {
			g.DealDamageToPermanent(atk, blk.CurrentPower(g), blk.ID())
		}
	}
}

// doBandedAttackDamage handles combat for an entire attacking band. It collects
// blockers from all band members' groups and processes them together.
func (c *Combat) doBandedAttackDamage(g *Game, bandMemberIDs []uuid.UUID, defenderID uuid.UUID, isFirstStrikeStep bool) {
	// Collect all blockers across all band members' groups (deduplicated).
	seen := make(map[uuid.UUID]bool)
	var allBlockerIDs []uuid.UUID
	for _, memberID := range bandMemberIDs {
		grp := c.GroupFor(memberID)
		if grp == nil {
			continue
		}
		for _, bid := range grp.BlockerIDs {
			if !seen[bid] {
				seen[bid] = true
				allBlockerIDs = append(allBlockerIDs, bid)
			}
		}
	}

	isBlocked := len(allBlockerIDs) > 0

	// Does the band contain any member with trample?
	hasTrample := false
	for _, memberID := range bandMemberIDs {
		member := g.FindPermanent(memberID)
		if member != nil && member.HasKeyword(Trample) {
			hasTrample = true
			break
		}
	}

	if !isBlocked {
		// Unblocked: each member independently deals its power to the defending player.
		for _, memberID := range bandMemberIDs {
			member := g.FindPermanent(memberID)
			if member == nil {
				continue
			}
			if !c.DealsDamageInStep(member, isFirstStrikeStep) {
				continue
			}
			dmg := member.CurrentPower(g)
			if dmg <= 0 {
				continue
			}
			defender := g.GetPlayer(defenderID)
			if defender == nil {
				continue
			}
			if dmg > 1 && g.Effects.Damage.HasForcefieldShield(defenderID) {
				dmg = 1
			}
			if bgID := g.Effects.Damage.GetBodyguard(defenderID); bgID != uuid.Nil {
				bg := g.FindPermanent(bgID)
				if bg != nil && !bg.Tapped {
					g.DealDamageToPermanent(bg, dmg, member.ID())
					continue
				}
			}
			g.DealDamageToPlayer(defender, dmg, member.ID())
		}
		return
	}

	// Blocked band.

	// 1. Band's total power (from members that deal damage in this step) goes to blockers.
	totalBandPower := 0
	var primaryAttacker *Permanent
	for _, memberID := range bandMemberIDs {
		member := g.FindPermanent(memberID)
		if member == nil {
			continue
		}
		if c.DealsDamageInStep(member, isFirstStrikeStep) {
			totalBandPower += member.CurrentPower(g)
			if primaryAttacker == nil {
				primaryAttacker = member
			}
		}
	}
	if primaryAttacker != nil && totalBandPower > 0 {
		remainingDmg := totalBandPower
		for _, bid := range allBlockerIDs {
			blk := g.FindPermanent(bid)
			if blk == nil {
				continue
			}
			needed := blk.CurrentToughness(g) - blk.Damage
			if needed <= 0 {
				continue
			}
			dealt := min(remainingDmg, needed)
			g.DealDamageToPermanent(blk, dealt, primaryAttacker.ID())
			remainingDmg -= dealt
			if remainingDmg <= 0 {
				break
			}
		}
		if remainingDmg > 0 && hasTrample {
			defender := g.GetPlayer(defenderID)
			if defender != nil {
				g.DealDamageToPlayer(defender, remainingDmg, primaryAttacker.ID())
			}
		}
	}

	// 2. Blockers' total damage is distributed by the attacking player across band members.
	totalIncoming := 0
	for _, bid := range allBlockerIDs {
		blk := g.FindPermanent(bid)
		if blk != nil && c.DealsDamageInStep(blk, isFirstStrikeStep) {
			totalIncoming += blk.CurrentPower(g)
		}
	}

	if totalIncoming > 0 {
		var attackingPlayerID uuid.UUID
		for _, memberID := range bandMemberIDs {
			member := g.FindPermanent(memberID)
			if member != nil {
				attackingPlayerID = member.Controller
				break
			}
		}
		attackingPlayer := g.GetPlayer(attackingPlayerID)

		var memberPerms []*Permanent
		for _, memberID := range bandMemberIDs {
			member := g.FindPermanent(memberID)
			if member != nil {
				memberPerms = append(memberPerms, member)
			}
		}

		var distribution map[uuid.UUID]int
		if distributor, ok := attackingPlayer.(BandingDamageDistributor); ok {
			distribution = distributor.GetBandingDamageDistribution(memberPerms)
		}

		sourceID := allBlockerIDs[0] // use first blocker as damage source for tracking
		if distribution != nil {
			for memberID, dmg := range distribution {
				if dmg <= 0 {
					continue
				}
				member := g.FindPermanent(memberID)
				if member != nil {
					g.DealDamageToPermanent(member, dmg, sourceID)
				}
			}
		} else {
			// Default: all incoming damage falls on the first band member.
			if len(memberPerms) > 0 {
				g.DealDamageToPermanent(memberPerms[0], totalIncoming, sourceID)
			}
		}
	}
}
