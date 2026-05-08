package mage

import (
	"slices"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// Combat manages the combat phase.
type Combat struct {
	Groups      []*CombatGroup
	Attackers   map[uuid.UUID]bool
	FirstStruck map[uuid.UUID]bool
	Bands       map[uuid.UUID][]uuid.UUID // band leader -> members (all creatures in the band)

	// AttackedAlone records the single creature declared as an attacker during
	// the most recent declare-attackers step, if exactly one was declared. Used
	// by the CR 506.5 "attacks alone" selector (e.g. Exalted, CR 702.83). Set
	// by SnapshotAttackedAlone, cleared by Reset.
	AttackedAlone uuid.UUID
	// BlockedAlone is the symmetric snapshot for CR 506.5 "blocks alone".
	BlockedAlone uuid.UUID
}

// CombatGroup represents an attacker and its blockers.
type CombatGroup struct {
	AttackerID uuid.UUID
	BlockerIDs []uuid.UUID
	DefenderID uuid.UUID
	Blocked    bool
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
	c.AttackedAlone = uuid.Nil
	c.BlockedAlone = uuid.Nil
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
			g.Blocked = true
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
		if slices.Contains(g.BlockerIDs, id) {
			return true
		}
	}
	return false
}

// SnapshotAttackedAlone records, at the end of the declare-attackers step,
// whether exactly one creature was declared as an attacker. CR 506.5: "A
// creature attacks alone if it's the only creature declared as an attacker
// during the declare attackers step."
func (c *Combat) SnapshotAttackedAlone() {
	if len(c.Attackers) == 1 {
		for id := range c.Attackers {
			c.AttackedAlone = id
		}
	} else {
		c.AttackedAlone = uuid.Nil
	}
}

// SnapshotBlockedAlone records, at the end of the declare-blockers step,
// whether exactly one creature was declared as a blocker. CR 506.5: "A
// creature blocks alone if it's the only creature declared as a blocker
// during the declare blockers step."
func (c *Combat) SnapshotBlockedAlone() {
	var only uuid.UUID
	count := 0
	for _, g := range c.Groups {
		for _, bid := range g.BlockerIDs {
			only = bid
			count++
			if count > 1 {
				c.BlockedAlone = uuid.Nil
				return
			}
		}
	}
	if count == 1 {
		c.BlockedAlone = only
	} else {
		c.BlockedAlone = uuid.Nil
	}
}

// AttacksAlone returns true if id was the sole creature declared as an
// attacker during the most recent declare-attackers step (CR 506.5,
// snapshot semantics — used by Exalted, etc.).
func (c *Combat) AttacksAlone(id uuid.UUID) bool {
	return id != uuid.Nil && c.AttackedAlone == id
}

// IsAttackingAlone returns true if id is currently the only attacker
// (CR 506.5, live semantics — re-evaluated whenever queried).
func (c *Combat) IsAttackingAlone(id uuid.UUID) bool {
	return c.Attackers[id] && len(c.Attackers) == 1
}

// BlocksAlone returns true if id was the sole creature declared as a
// blocker during the most recent declare-blockers step (CR 506.5,
// snapshot semantics).
func (c *Combat) BlocksAlone(id uuid.UUID) bool {
	return id != uuid.Nil && c.BlockedAlone == id
}

// IsBlockingAlone returns true if id is currently the only blocker
// across all combat groups (CR 506.5, live semantics).
func (c *Combat) IsBlockingAlone(id uuid.UUID) bool {
	found := false
	count := 0
	for _, g := range c.Groups {
		for _, bid := range g.BlockerIDs {
			count++
			if bid == id {
				found = true
			}
			if count > 1 {
				return false
			}
		}
	}
	return found && count == 1
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
	return slices.Contains(members, id2)
}

// RemoveFromCombat removes a permanent from combat (attacker or blocker).
// Per Oracle text on cards like False Orders and Ydwen Efreet: "Creatures it
// was blocking that had become blocked by only that creature this combat become
// unblocked." If the removed creature was the sole blocker, the attacker
// becomes unblocked (Blocked = false).
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
		// If this was the only blocker, the attacker becomes unblocked.
		if len(g.BlockerIDs) == 0 {
			g.Blocked = false
		}
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
	if g.effects.IsBlockPrevented(blocker.ID(), attacker.ID()) {
		return false
	}
	// Filter-based "can't be blocked except by X" / "can block only X" rules
	// (CR 509.1b). Source: combat_restrictions.go.
	if !g.effects.passesCombatRestrictions(blocker, attacker, g) {
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
		isBlack := slices.Contains(blocker.Card.ManaCost().Colors(), Black)
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
		if attacker.HasKeyword(kw) && !g.effects.Rules.IsLandwalkNullified(kw) {
			for _, p := range g.battlefield {
				if p.Controller == defenderID && p.HasType(TypeLand) && p.HasSubType(subtype) {
					return true
				}
			}
		}
	}
	// Check legendary landwalk: unblockable if defender controls a legendary land
	if attacker.HasKeyword(LegendaryLandwalk) {
		for _, p := range g.battlefield {
			if p.Controller == defenderID && p.HasType(TypeLand) && p.Card.HasSuperType(SuperLegendary) {
				return true
			}
		}
	}
	return false
}

// combatDamageValue returns the amount of damage a creature assigns in combat:
// its toughness if AttrAssignsDamageEqualToToughness is set (Doran the Siege
// Tower, Assault Formation), otherwise its power. CR 702.x: such effects do
// not change the creature's power; they only change the value used for combat
// damage assignment.
func combatDamageValue(p *Permanent, g *Game) int {
	if p.HasAttr(AttrAssignsDamageEqualToToughness) {
		return p.CurrentToughness(g)
	}
	return p.CurrentPower(g)
}

// ResolveDamage executes combat damage for all combat groups. It replaces the
// old Game.DoCombatDamage method and operates on the Combat receiver so the
// logic lives alongside the rest of the combat state.
func (c *Combat) ResolveDamage(g *Game, isFirstStrikeStep bool) {
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
		if !group.Blocked {
			// Unblocked — damage to defending player.
			// Forcefield and bodyguard are now replacement effects in the pipeline.
			if c.DealsDamageInStep(atk, isFirstStrikeStep) {
				defender := g.GetPlayer(group.DefenderID)
				if defender != nil {
					dmg := combatDamageValue(atk, g)
					g.DealDamageToPlayer(defender, dmg, atk.ID())
				}
			}
		} else if len(group.BlockerIDs) == 0 {
			// CR 509.1h + 510.1c: blocked but all blockers removed.
			// Deals no damage unless it has trample.
			if atk.HasKeyword(Trample) && c.DealsDamageInStep(atk, isFirstStrikeStep) {
				defender := g.GetPlayer(group.DefenderID)
				if defender != nil {
					dmg := atk.CurrentPower(g)
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
		atkPower := combatDamageValue(atk, g)
		// Order: ask attacker controller (CR 510.1c) for the damage-assignment
		// order. Default is the BlockerIDs order recorded at block declaration.
		orderedIDs := group.BlockerIDs
		attackingPlayer := g.GetPlayer(atk.Controller)
		blockerPerms := make([]*Permanent, 0, len(group.BlockerIDs))
		for _, bid := range group.BlockerIDs {
			if blk := g.FindPermanent(bid); blk != nil {
				blockerPerms = append(blockerPerms, blk)
			}
		}
		if assigner, ok := attackingPlayer.(CombatDamageAssigner); ok && len(blockerPerms) > 0 {
			if reordered := assigner.GetBlockerOrder(atk, blockerPerms); reordered != nil {
				orderedIDs = mergeBlockerOrder(reordered, group.BlockerIDs)
			}
		}

		// Try a controller-supplied damage assignment first; fall back to the
		// engine's lethal-first greedy split if none is provided or it is
		// invalid per CR 510.1c (each blocker before the next must be assigned
		// at least lethal damage; with trample, all blockers must be at lethal
		// before any goes to the defender).
		var assignment map[uuid.UUID]int
		if assigner, ok := attackingPlayer.(CombatDamageAssigner); ok && len(orderedIDs) > 1 {
			orderedPerms := make([]*Permanent, 0, len(orderedIDs))
			for _, bid := range orderedIDs {
				if blk := g.FindPermanent(bid); blk != nil {
					orderedPerms = append(orderedPerms, blk)
				}
			}
			assignment = assigner.GetCombatDamageAssignment(atk, orderedPerms, atkPower)
			if assignment != nil && !validateBlockerAssignment(g, atk, orderedIDs, assignment, atkPower) {
				assignment = nil
			}
		}

		remainingDmg := atkPower
		if assignment != nil {
			usedDmg := 0
			for _, bid := range orderedIDs {
				blk := g.FindPermanent(bid)
				if blk == nil {
					continue
				}
				dmg := assignment[bid]
				if dmg > 0 {
					g.DealDamageToPermanent(blk, dmg, atk.ID())
					usedDmg += dmg
				}
			}
			remainingDmg = atkPower - usedDmg
		} else {
			for _, bid := range orderedIDs {
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
			g.DealDamageToPermanent(atk, combatDamageValue(blk, g), blk.ID())
		}
	}
}

// mergeBlockerOrder returns a permutation of original such that IDs in
// preferred come first (in their preferred order), then any IDs from original
// not present in preferred (in their original order).
func mergeBlockerOrder(preferred, original []uuid.UUID) []uuid.UUID {
	origSet := make(map[uuid.UUID]bool, len(original))
	for _, id := range original {
		origSet[id] = true
	}
	seen := make(map[uuid.UUID]bool, len(original))
	out := make([]uuid.UUID, 0, len(original))
	for _, id := range preferred {
		if origSet[id] && !seen[id] {
			out = append(out, id)
			seen[id] = true
		}
	}
	for _, id := range original {
		if !seen[id] {
			out = append(out, id)
			seen[id] = true
		}
	}
	return out
}

// validateBandedBlockerAssignment is the banding-relaxed counterpart of
// validateBlockerAssignment. Per CR 702.22, the attacking player may
// distribute the band's combined damage among the blockers freely, so the
// lethal-first ordering of CR 510.1c does not apply. Total must equal the
// attacker's power; with trample, total may be less only if every blocker
// has been assigned at least lethal damage.
func validateBandedBlockerAssignment(g *Game, blockers []*Permanent, assignment map[uuid.UUID]int, totalPower int, hasTrample bool) bool {
	total := 0
	allLethal := true
	for _, blk := range blockers {
		if blk == nil {
			continue
		}
		dmg := assignment[blk.ID()]
		if dmg < 0 {
			return false
		}
		total += dmg
		needed := max(blk.CurrentToughness(g)-blk.Damage, 0)
		if dmg < needed {
			allLethal = false
		}
	}
	if total > totalPower {
		return false
	}
	if total < totalPower && (!hasTrample || !allLethal) {
		return false
	}
	return true
}

// validateBlockerAssignment checks the CR 510.1c constraint: damage is
// assigned to blockers in order, and a blocker can only be assigned non-lethal
// damage if every blocker before it in the order has been assigned at least
// lethal damage. With trample (CR 702.19b) all blockers must be at lethal
// before any goes to the defender. The total assigned must not exceed
// atkPower.
func validateBlockerAssignment(g *Game, atk *Permanent, orderedIDs []uuid.UUID, assignment map[uuid.UUID]int, atkPower int) bool {
	total := 0
	prevLethal := true
	allLethal := true
	for _, bid := range orderedIDs {
		blk := g.FindPermanent(bid)
		if blk == nil {
			continue
		}
		needed := max(blk.CurrentToughness(g)-blk.Damage, 0)
		dmg := assignment[bid]
		if dmg < 0 {
			return false
		}
		total += dmg
		if !prevLethal && dmg > 0 {
			return false
		}
		if dmg < needed {
			prevLethal = false
			allLethal = false
		}
	}
	if total > atkPower {
		return false
	}
	if total < atkPower {
		if !atk.HasKeyword(Trample) || !allLethal {
			return false
		}
	}
	return true
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
		atkPower := combatDamageValue(atk, g)

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
			g.DealDamageToPermanent(atk, combatDamageValue(blk, g), blk.ID())
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

	// Check if any member of the band was blocked (CR 509.1h).
	isBlocked := false
	for _, memberID := range bandMemberIDs {
		grp := c.GroupFor(memberID)
		if grp != nil && grp.Blocked {
			isBlocked = true
			break
		}
	}

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
			dmg := combatDamageValue(member, g)
			if dmg <= 0 {
				continue
			}
			defender := g.GetPlayer(defenderID)
			if defender == nil {
				continue
			}
			// Forcefield and bodyguard are now replacement effects in the pipeline.
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
			totalBandPower += combatDamageValue(member, g)
			if primaryAttacker == nil {
				primaryAttacker = member
			}
		}
	}
	if primaryAttacker != nil && totalBandPower > 0 {
		// CR 702.22: the attacking player chooses how the band's combined
		// damage is distributed among blockers, ignoring the lethal-first
		// ordering of CR 510.1c. Consult CombatDamageAssigner; if absent or
		// invalid, fall back to greedy lethal-first.
		blockerPerms := make([]*Permanent, 0, len(allBlockerIDs))
		for _, bid := range allBlockerIDs {
			if blk := g.FindPermanent(bid); blk != nil {
				blockerPerms = append(blockerPerms, blk)
			}
		}
		attackingPlayer := g.GetPlayer(primaryAttacker.Controller)
		var assignment map[uuid.UUID]int
		if assigner, ok := attackingPlayer.(CombatDamageAssigner); ok && len(blockerPerms) > 0 {
			assignment = assigner.GetCombatDamageAssignment(primaryAttacker, blockerPerms, totalBandPower)
			if assignment != nil && !validateBandedBlockerAssignment(g, blockerPerms, assignment, totalBandPower, hasTrample) {
				assignment = nil
			}
		}

		remainingDmg := totalBandPower
		if assignment != nil {
			usedDmg := 0
			for _, blk := range blockerPerms {
				if dmg := assignment[blk.ID()]; dmg > 0 {
					g.DealDamageToPermanent(blk, dmg, primaryAttacker.ID())
					usedDmg += dmg
				}
			}
			remainingDmg = totalBandPower - usedDmg
		} else {
			for _, blk := range blockerPerms {
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
			totalIncoming += combatDamageValue(blk, g)
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
		} else if len(memberPerms) > 0 {
			// Default: all incoming damage falls on the first band member.
			g.DealDamageToPermanent(memberPerms[0], totalIncoming, sourceID)
		}
	}
}
