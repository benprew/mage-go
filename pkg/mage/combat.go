package mage

import (
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// Combat manages the combat phase.
type Combat struct {
	assignmentEligibility map[capturedPermanent]bool
	Groups                []*CombatGroup
	Attackers             map[uuid.UUID]bool
	AttackerIncarnations  map[uuid.UUID]uuid.UUID
	FirstStruck           map[uuid.UUID]bool
	Bands                 map[uuid.UUID][]uuid.UUID // band leader -> members (all creatures in the band)

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
	AttackerID          uuid.UUID
	BlockerIDs          []uuid.UUID
	BlockerIncarnations []uuid.UUID
	DefenderID          uuid.UUID
	Blocked             bool
	Unblocked           bool
	blockedBy           map[capturedPermanent]bool
}

func NewCombat() *Combat {
	return &Combat{
		Attackers:   make(map[uuid.UUID]bool),
		FirstStruck: make(map[uuid.UUID]bool),
		Bands:       make(map[uuid.UUID][]uuid.UUID),
	}
}

func (c *Combat) Reset() {
	c.assignmentEligibility = nil
	c.Groups = nil
	c.Attackers = make(map[uuid.UUID]bool)
	c.AttackerIncarnations = nil
	c.FirstStruck = make(map[uuid.UUID]bool)
	c.Bands = make(map[uuid.UUID][]uuid.UUID)
	c.AttackedAlone = uuid.Nil
	c.BlockedAlone = uuid.Nil
}

func (c *Combat) AddAttacker(attackerID, defenderID uuid.UUID) {
	c.AddAttackerIncarnation(attackerID, uuid.Nil, defenderID)
}

// AddAttackerIncarnation records the battlefield object that attacks.
func (c *Combat) AddAttackerIncarnation(attackerID, incarnation, defenderID uuid.UUID) {
	c.Attackers[attackerID] = true
	if c.AttackerIncarnations == nil {
		c.AttackerIncarnations = make(map[uuid.UUID]uuid.UUID)
	}
	c.AttackerIncarnations[attackerID] = incarnation
	c.Groups = append(c.Groups, &CombatGroup{
		AttackerID: attackerID,
		DefenderID: defenderID,
	})
}

// IsAttackingIncarnation checks the object that entered combat.
func (c *Combat) IsAttackingIncarnation(id, incarnation uuid.UUID) bool {
	return incarnation != uuid.Nil && c.Attackers[id] && c.AttackerIncarnations[id] == incarnation
}

func (c *Combat) AddBlocker(blockerID, attackerID uuid.UUID) {
	c.AddBlockerIncarnation(blockerID, uuid.Nil, attackerID)
}

// AddBlockerIncarnation records the battlefield object that blocks.
func (c *Combat) AddBlockerIncarnation(blockerID, incarnation, attackerID uuid.UUID) {
	for _, g := range c.Groups {
		if g.AttackerID != attackerID && !c.IsBandedWith(g.AttackerID, attackerID) {
			continue
		}
		if slices.Contains(g.BlockerIDs, blockerID) {
			continue
		}
		g.BlockerIDs = append(g.BlockerIDs, blockerID)
		g.BlockerIncarnations = append(g.BlockerIncarnations, incarnation)
		g.Blocked = true
		g.Unblocked = false
		if g.blockedBy == nil {
			g.blockedBy = make(map[capturedPermanent]bool)
		}
		g.blockedBy[capturedPermanent{blockerID, incarnation}] = true
	}
}

// IsUnblockedIncarnation requires the original attacker after blocker declaration.
func (c *Combat) IsUnblockedIncarnation(id, incarnation uuid.UUID) bool {
	if !c.IsAttackingIncarnation(id, incarnation) {
		return false
	}
	group := c.GroupFor(id)
	return group != nil && group.Unblocked && !group.Blocked
}

// IsBlockingIncarnation checks the object that entered combat.
func (c *Combat) IsBlockingIncarnation(id, incarnation uuid.UUID) bool {
	if incarnation == uuid.Nil {
		return false
	}
	for _, group := range c.Groups {
		for i, blockerID := range group.BlockerIDs {
			if blockerID == id && i < len(group.BlockerIncarnations) && group.BlockerIncarnations[i] == incarnation {
				return true
			}
		}
	}
	return false
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
	seen := make(map[uuid.UUID]bool)
	for _, g := range c.Groups {
		for _, bid := range g.BlockerIDs {
			if seen[bid] {
				continue
			}
			seen[bid] = true
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

// RemoveFromCombat removes a permanent without changing blocked status.
func (c *Combat) RemoveFromCombat(id uuid.UUID) {
	delete(c.Attackers, id)
	delete(c.AttackerIncarnations, id)
	if members, ok := c.Bands[id]; ok {
		remaining := make([]uuid.UUID, 0, len(members))
		for _, member := range members {
			if member != id {
				remaining = append(remaining, member)
			}
		}
		delete(c.Bands, id)
		for _, member := range remaining {
			c.Bands[member] = remaining
		}
	}
	for _, g := range c.Groups {
		if g.AttackerID == id {
			g.AttackerID = uuid.Nil
		}
		filtered := g.BlockerIDs[:0]
		incarnations := make([]uuid.UUID, 0, len(g.BlockerIDs))
		for i, bid := range g.BlockerIDs {
			if bid != id {
				filtered = append(filtered, bid)
				if i < len(g.BlockerIncarnations) {
					incarnations = append(incarnations, g.BlockerIncarnations[i])
				} else {
					incarnations = append(incarnations, uuid.Nil)
				}
			}
		}
		g.BlockerIDs = filtered
		g.BlockerIncarnations = incarnations
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

// Strike eligibility must remain recorded when assignment is suppressed.
func (c *Combat) assignsDamageInStep(g *Game, p *Permanent, firstStrike bool) bool {
	if c.assignmentEligibility != nil {
		return c.assignmentEligibility[capturedPermanent{p.ID(), p.incarnationID}] && g.effects.noCombatAssignmentRules[p.ID()] != p.incarnationID
	}
	eligible := c.DealsDamageInStep(p, firstStrike)
	return eligible && g.effects.noCombatAssignmentRules[p.ID()] != p.incarnationID
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
	if attacker.HasProtectionFromInGame(blocker.Card, g) {
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
		isBlack := slices.Contains(blocker.Colors(), Black)
		if !isArtifact && !isBlack {
			return false
		}
	}
	// CantBeBlockedExceptByWalls: can only be blocked by Walls (e.g. Invisibility)
	if attacker.HasKeyword(CantBeBlockedExceptByWalls) && !blocker.HasSubType("Wall") {
		return false
	}
	// The declaration service checks the number of blockers for menace.
	return true
}

// HasLandwalkEvasion returns true if the attacker has a landwalk ability
// and the defending player controls a land of the matching subtype.
// Uses the data-driven landwalkSubtypes map so new landwalk variants
// (e.g. Desertwalk) work automatically.
func HasLandwalkEvasion(attacker *Permanent, defenderID uuid.UUID, g *Game) bool {
	for kw, subtype := range LandwalkAttrs() {
		if attacker.HasKeyword(kw) && !g.effects.Rules.IsLandwalkNullified(kw) {
			for _, p := range g.zones.battlefield {
				if !p.PhasedOut && p.ControllerID() == defenderID && p.HasType(TypeLand) && p.HasSubType(subtype) {
					return true
				}
			}
		}
	}
	// Check legendary landwalk: unblockable if defender controls a legendary land
	if attacker.HasKeyword(LegendaryLandwalk) {
		for _, p := range g.zones.battlefield {
			if !p.PhasedOut && p.ControllerID() == defenderID && p.HasType(TypeLand) && p.Card.HasSuperType(SuperLegendary) {
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

type combatDamagePlan struct{ assignments []DamageAssignment }

func (p *combatDamagePlan) player(g *Game, recipient Player, amount int, sourceID uuid.UUID) {
	p.assignments = append(p.assignments, PlayerDamageAssignment(recipient, amount, g.CaptureDamageSource(sourceID)))
}
func (p *combatDamagePlan) permanent(g *Game, recipient *Permanent, amount int, sourceID uuid.UUID) {
	p.assignments = append(p.assignments, PermanentDamageAssignment(recipient, amount, g.CaptureDamageSource(sourceID)))
}

// ResolveDamage executes combat damage for all combat groups. It replaces the
// old Game.DoCombatDamage method and operates on the Combat receiver so the
// logic lives alongside the rest of the combat state.
func (c *Combat) ResolveDamage(g *Game, isFirstStrikeStep bool) {
	previousEligibility := c.assignmentEligibility
	c.assignmentEligibility = make(map[capturedPermanent]bool)
	defer func() { c.assignmentEligibility = previousEligibility }()
	c.prepareFaceUpAssignments(g, isFirstStrikeStep)
	if c.needsSharedBandAssignment(g) {
		c.resolveSharedBandDamage(g, isFirstStrikeStep)
		return
	}
	// Track which band leaders we've already processed to avoid double-dealing.
	processedBands := make(map[uuid.UUID]bool)
	// Save all assignments before replacement choices can change game state.
	plan := &combatDamagePlan{}

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
			c.doBandedAttackDamage(g, plan, bandMembers, group.DefenderID, isFirstStrikeStep)
			continue
		}

		// Non-banded attacker.
		if !group.Blocked {
			// Unblocked — damage to defending player.
			// Forcefield and bodyguard are now replacement effects in the pipeline.
			if c.assignsDamageInStep(g, atk, isFirstStrikeStep) {
				defender := g.GetPlayer(group.DefenderID)
				if defender != nil {
					dmg := combatDamageValue(atk, g)
					plan.player(g, defender, dmg, atk.ID())
				}
			}
		} else if len(group.BlockerIDs) == 0 {
			// CR 509.1h + 510.1c: blocked but all blockers removed.
			// Deals no damage unless it has trample.
			if atk.HasKeyword(Trample) && c.assignsDamageInStep(g, atk, isFirstStrikeStep) {
				defender := g.GetPlayer(group.DefenderID)
				if defender != nil {
					dmg := atk.CurrentPower(g)
					plan.player(g, defender, dmg, atk.ID())
				}
			}
		} else if c.isBlockingBand(g, group.BlockerIDs) {
			// Blocking band: defending player distributes attacker's damage.
			c.doBlockingBandDamage(g, plan, atk, group, isFirstStrikeStep)
		} else {
			// Normal blocked combat.
			c.doNormalBlockedDamage(g, plan, atk, group, isFirstStrikeStep)
		}
	}

	g.DealSimultaneousDamage(plan.assignments...)
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
func (c *Combat) doNormalBlockedDamage(g *Game, plan *combatDamagePlan, atk *Permanent, group *CombatGroup, isFirstStrikeStep bool) {
	if !atk.HasKeyword(Trample) {
		c.doNontrampleBlockedDamage(g, plan, atk, group, isFirstStrikeStep)
		return
	}
	if c.assignsDamageInStep(g, atk, isFirstStrikeStep) {
		atkPower := combatDamageValue(atk, g)
		// Order: ask attacker controller (CR 510.1c) for the damage-assignment
		// order. Default is the BlockerIDs order recorded at block declaration.
		orderedIDs := group.BlockerIDs
		attackingPlayer := g.GetPlayer(atk.ControllerID())
		blockerPerms := make([]*Permanent, 0, len(group.BlockerIDs))
		for _, bid := range group.BlockerIDs {
			if blk := g.FindPermanent(bid); blk != nil {
				blockerPerms = append(blockerPerms, blk)
			}
		}
		if assigner, ok := attackingPlayer.(CombatDamageAssigner); ok && len(blockerPerms) > 0 {
			if reordered := assigner.GetBlockerOrder(g, atk, blockerPerms, atkPower); reordered != nil {
				orderedIDs = mergeBlockerOrder(reordered, group.BlockerIDs)
			}
		}

		// Try a controller-supplied damage assignment first; fall back to the
		// engine's lethal-first greedy split if none is provided or it is
		// invalid per CR 510.1c (each blocker before the next must be assigned
		// at least lethal damage; with trample, all blockers must be at lethal
		// before any goes to the defender).
		orderedPerms := make([]*Permanent, 0, len(orderedIDs))
		for _, bid := range orderedIDs {
			if blk := g.FindPermanent(bid); blk != nil {
				orderedPerms = append(orderedPerms, blk)
			}
		}
		var assignment map[uuid.UUID]int
		if assigner, ok := attackingPlayer.(CombatDamageAssigner); ok && len(orderedIDs) > 1 {
			assignment = assigner.GetCombatDamageAssignment(g, atk, orderedPerms, atkPower)
			if assignment != nil && !validateBlockerAssignment(g, atk, orderedIDs, assignment, atkPower) {
				assignment = nil
			}
		}

		var remainingDmg int
		if assignment != nil {
			usedDmg := 0
			for _, bid := range orderedIDs {
				blk := g.FindPermanent(bid)
				if blk == nil {
					continue
				}
				dmg := assignment[bid]
				if dmg > 0 {
					plan.permanent(g, blk, dmg, atk.ID())
					usedDmg += dmg
				}
			}
			remainingDmg = atkPower - usedDmg
		} else {
			remainingDmg = dealDefaultBlockedCombatDamage(g, plan, atk, orderedPerms, atkPower, atk.HasKeyword(Trample))
		}
		if remainingDmg > 0 && atk.HasKeyword(Trample) {
			defender := g.GetPlayer(group.DefenderID)
			if defender != nil {
				plan.player(g, defender, remainingDmg, atk.ID())
			}
		}
	}
	for _, bid := range group.BlockerIDs {
		blk := g.FindPermanent(bid)
		if blk == nil {
			continue
		}
		if c.assignsDamageInStep(g, blk, isFirstStrikeStep) {
			plan.permanent(g, atk, combatDamageValue(blk, g), blk.ID())
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
		needed := lethalCombatDamageRequired(atk, blk, g)
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

func lethalCombatDamageRequired(source, target *Permanent, g *Game) int {
	if source == nil || target == nil {
		return 0
	}
	needed := max(target.CurrentToughness(g)-target.Damage, 0)
	if source.HasKeyword(Deathtouch) && needed > 0 {
		return 1
	}
	return needed
}

func dealDefaultBlockedCombatDamage(g *Game, plan *combatDamagePlan, source *Permanent, blockers []*Permanent, total int, hasTrample bool) int {
	remaining := total
	for i, blocker := range blockers {
		dealt := min(remaining, lethalCombatDamageRequired(source, blocker, g))
		if !hasTrample && i == len(blockers)-1 {
			dealt = remaining
		}
		if dealt > 0 {
			plan.permanent(g, blocker, dealt, source.ID())
			remaining -= dealt
		}
		if remaining <= 0 {
			break
		}
	}
	return remaining
}

// doBlockingBandDamage handles combat where multiple blockers with banding block
// a single attacker. The defending player (controller of the blocking band) chooses
// how the attacker's damage is distributed across band members.
func (c *Combat) doBlockingBandDamage(g *Game, plan *combatDamagePlan, atk *Permanent, group *CombatGroup, isFirstStrikeStep bool) {
	// Collect blocker permanents.
	var blockerPerms []*Permanent
	for _, bid := range group.BlockerIDs {
		blk := g.FindPermanent(bid)
		if blk != nil {
			blockerPerms = append(blockerPerms, blk)
		}
	}

	// Attacker deals damage — defending player distributes it across the blocking band.
	if c.assignsDamageInStep(g, atk, isFirstStrikeStep) {
		atkPower := combatDamageValue(atk, g)

		var defendingPlayerID uuid.UUID
		if len(blockerPerms) > 0 {
			defendingPlayerID = blockerPerms[0].ControllerID()
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
					plan.permanent(g, blk, dmg, atk.ID())
					usedDmg += dmg
				}
			}
			// Trample: any damage beyond what was distributed goes to the defending player.
			if atk.HasKeyword(Trample) {
				if trampleDmg := atkPower - usedDmg; trampleDmg > 0 {
					defender := g.GetPlayer(group.DefenderID)
					if defender != nil {
						plan.player(g, defender, trampleDmg, atk.ID())
					}
				}
			}
		} else {
			// Default: normal distribution (no banding benefit).
			remainingDmg := dealDefaultBlockedCombatDamage(g, plan, atk, blockerPerms, atkPower, atk.HasKeyword(Trample))
			if remainingDmg > 0 && atk.HasKeyword(Trample) {
				defender := g.GetPlayer(group.DefenderID)
				if defender != nil {
					plan.player(g, defender, remainingDmg, atk.ID())
				}
			}
		}
	}

	// Each blocker still deals its own damage to the attacker.
	for _, blk := range blockerPerms {
		if c.assignsDamageInStep(g, blk, isFirstStrikeStep) {
			plan.permanent(g, atk, combatDamageValue(blk, g), blk.ID())
		}
	}
}

// doBandedAttackDamage handles combat for an entire attacking band. It collects
// blockers from all band members' groups and processes them together.
func (c *Combat) doBandedAttackDamage(g *Game, plan *combatDamagePlan, bandMemberIDs []uuid.UUID, defenderID uuid.UUID, isFirstStrikeStep bool) {
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

	if !isBlocked {
		// Unblocked: each member independently deals its power to the defending player.
		for _, memberID := range bandMemberIDs {
			member := g.FindPermanent(memberID)
			if member == nil {
				continue
			}
			if !c.assignsDamageInStep(g, member, isFirstStrikeStep) {
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
			plan.player(g, defender, dmg, member.ID())
		}
		return
	}

	c.assignOutgoingBandDamage(g, plan, bandMemberIDs, allBlockerIDs, defenderID, isFirstStrikeStep)

	c.assignIncomingBandDamage(g, plan, bandMemberIDs, allBlockerIDs, isFirstStrikeStep)
}

// Legacy card effects also make attackers with only this blocker unblocked.
func (c *Combat) removeFromCombatAndUnblock(id uuid.UUID) {
	for _, group := range c.Groups {
		if len(group.BlockerIDs) == 1 && group.BlockerIDs[0] == id {
			group.Blocked = false
			group.Unblocked = true
		}
	}
	c.RemoveFromCombat(id)
}

// The current blocker list cannot establish which objects blocked this combat.
func (c *Combat) removeIncarnationAndUnblock(id, incarnation uuid.UUID) {
	for _, group := range c.Groups {
		if len(group.blockedBy) != 1 || !group.blockedBy[capturedPermanent{id, incarnation}] {
			continue
		}
		for i, blockerID := range group.BlockerIDs {
			if blockerID == id && i < len(group.BlockerIncarnations) && group.BlockerIncarnations[i] == incarnation {
				group.Blocked = false
				group.Unblocked = true
				break
			}
		}
	}
	c.RemoveFromCombat(id)
}

func snapshotCombatOpponents(source *Permanent, g GameReader) []capturedPermanent {
	var result []capturedPermanent
	seen := make(map[capturedPermanent]bool)
	add := func(id uuid.UUID) {
		p := g.FindPermanent(id)
		if p == nil || p.PhasedOut {
			return
		}
		ref := capturedPermanent{p.ID(), p.incarnationID}
		if !seen[ref] {
			seen[ref] = true
			result = append(result, ref)
		}
	}
	for _, group := range g.CombatGroups() {
		if group.AttackerID == source.ID() {
			for _, id := range group.BlockerIDs {
				add(id)
			}
		}
		for i, id := range group.BlockerIDs {
			if id == source.ID() && i < len(group.BlockerIncarnations) && group.BlockerIncarnations[i] == source.incarnationID {
				add(group.AttackerID)
			}
		}
	}
	return result
}

func (c *Combat) prepareFaceUpAssignments(g *Game, firstStrike bool) {
	seen := make(map[uuid.UUID]bool)
	var attackers, blockers []uuid.UUID
	for _, group := range c.Groups {
		attacker := g.FindPermanent(group.AttackerID)
		if attacker == nil || attacker.PhasedOut || !attacker.HasType(TypeCreature) {
			continue
		}
		if incarnation := c.AttackerIncarnations[attacker.ID()]; incarnation != uuid.Nil && incarnation != attacker.incarnationID {
			continue
		}
		c.assignmentEligibility[capturedPermanent{attacker.ID(), attacker.incarnationID}] = c.DealsDamageInStep(attacker, firstStrike)
		if !group.Blocked && !g.isValidDefender(group.DefenderID) {
			continue
		}
		if !group.Blocked || len(liveNormalBlockers(g, group)) > 0 || attacker.HasKeyword(Trample) {
			attackers = append(attackers, attacker.ID())
		}
		for _, blocker := range liveNormalBlockers(g, group) {
			c.assignmentEligibility[capturedPermanent{blocker.ID(), blocker.incarnationID}] = c.DealsDamageInStep(blocker, firstStrike)
			blockers = append(blockers, blocker.ID())
		}
	}
	for _, id := range append(attackers, blockers...) {
		if seen[id] {
			continue
		}
		seen[id] = true
		p := g.FindPermanent(id)
		if p == nil || p.PhasedOut || !p.HasType(TypeCreature) || combatDamageValue(p, g) <= 0 || !c.assignsDamageInStep(g, p, firstStrike) {
			continue
		}
		g.effects.ApplyReplacements(&assignDamageAction{capturedPermanent{p.ID(), p.incarnationID}}, g)
	}
}
