package interactive

import (
	"math"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// ThreatPerMana returns a permanent's threat-per-mana-spent ratio.
// Returns 0 if the permanent has CMC 0 (tokens, lands).
// Used by autoSelectTargets to prefer the most mana-efficient threat.
func ThreatPerMana(perm *mage.Permanent) float64 {
	cmc := perm.Card.ManaCost().CMC()
	if cmc == 0 {
		return 0
	}
	return float64(evalCreature(perm)) / float64(cmc)
}

// spellIsWorthless returns true if the spell requires at least one target but
// has no valid targets available right now. Used to skip casting a targeted
// sorcery when there is nothing to hit (avoids a main-phase cast bug where
// the AI commits to casting but then has nil targets).
func spellIsWorthless(card mage.Card, p mage.Player, g *mage.Game) bool {
	playerID := p.PlayerID()
	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok {
			continue
		}
		for _, t := range sa.Targets() {
			if len(t.Possible(playerID, card, g)) == 0 {
				return true
			}
		}
	}
	return false
}

// spellValue computes a context-sensitive score for a spell in the AI's hand,
// used to pick the best spell to cast each turn.
//
// Scoring rules (applied to each effect in the spell's SpellAbility):
//   - Creature:          basePower*2 + baseToughness (fallback: CMC)
//   - DrawCount > 0:     +DrawCount*3
//   - Damage > 0, detriment: +Damage, +2 per opponent creature it would kill
//   - Mass == true:      +20 if behind on board (DefaultEvaluator < 0), -10 if ahead
//   - Detriment, targeted, no damage: +evalCreature score of best reachable opponent permanent
//   - Fallback:          CMC
func spellValue(card mage.Card, p mage.Player, g *mage.Game) int {
	playerID := p.PlayerID()
	cmc := card.ManaCost().CMC()

	// Creatures: P/T heuristic
	if card.HasType(core.TypeCreature) {
		pw := card.Power()
		tg := card.Toughness()
		v := pw*2 + tg
		if v > 0 {
			return v
		}
		return cmc
	}

	score := 0
	found := false

	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok {
			continue
		}
		for _, e := range sa.Effects() {
			props := e.Properties()

			if props.DrawCount > 0 {
				score += props.DrawCount * 3
				found = true
			}

			if props.DamageValue != nil && props.Outcome == mage.OutcomeDetriment {
				dmg := props.DamageValue.Resolve(g, card.ID(), playerID)
				score += dmg
				// Bonus for lethal hits against opponent creatures
				opponent := g.GetOpponent(playerID)
				if opponent != nil {
					for _, perm := range g.Battlefield {
						if perm.Controller == opponent.PlayerID() && perm.HasType(core.TypeCreature) {
							if dmg >= perm.CurrentToughness(g) {
								score += 2
								break // count the lethality bonus once
							}
						}
					}
				}
				found = true
			}

			if props.Mass {
				if DefaultEvaluator(g, playerID) < 0 {
					score += 20
				} else {
					score -= 10
				}
				found = true
			}

			// Pure targeted removal (destroy, exile, bounce, etc.): value by best target
			if props.Outcome == mage.OutcomeDetriment && !props.Mass && props.DamageValue == nil {
				opponent := g.GetOpponent(playerID)
				if opponent != nil {
					bestTS := 0
					for _, perm := range g.Battlefield {
						if perm.Controller == opponent.PlayerID() {
							ts := evalCreature(perm)
							if ts > bestTS {
								bestTS = ts
							}
						}
					}
					if bestTS > 0 {
						score += bestTS
						found = true
					}
				}
			}
		}
	}

	if found && score > 0 {
		return score
	}
	return cmc
}

// ─── Lethal Detection (Phase 0A) ────────────────────────────────────────────

// LethalInfo describes whether either player can push through lethal damage
// on their next attack step.
type LethalInfo struct {
	IHaveLethal      bool
	TheyHaveLethal   bool
	MyBoardDamage    int         // total damage I can push through
	TheirBoardDamage int         // total damage they can push through
	LethalAttackers  []uuid.UUID // minimal set that kills opponent
}

// CalculateLethal computes lethal-on-board information for playerID.
// It estimates how much damage each side can push through blockers by
// categorizing attackers as evasive (flying, unblockable, fear, landwalk,
// menace) or non-evasive, and accounting for trample.
func CalculateLethal(g *mage.Game, playerID uuid.UUID) LethalInfo {
	opponent := g.GetOpponent(playerID)
	if opponent == nil {
		return LethalInfo{}
	}
	oppID := opponent.PlayerID()

	myDmg, myLethal := estimatePushThroughDamage(g, playerID, oppID)
	theirDmg, _ := estimatePushThroughDamage(g, oppID, playerID)

	info := LethalInfo{
		MyBoardDamage:    myDmg,
		TheirBoardDamage: theirDmg,
	}

	if myDmg >= opponent.Life() {
		info.IHaveLethal = true
		info.LethalAttackers = findMinimalLethalSet(g, playerID, oppID, opponent.Life())
		// If minimal set is empty but total is lethal, fall back to all eligible
		if len(info.LethalAttackers) == 0 {
			info.LethalAttackers = myLethal
		}
	}

	me := g.GetPlayer(playerID)
	if me != nil && theirDmg >= me.Life() {
		info.TheyHaveLethal = true
	}

	return info
}

// isEvasive returns true if the attacker has evasion that lets it bypass most blockers.
func isEvasive(perm *mage.Permanent, defenderID uuid.UUID, g *mage.Game) bool {
	if perm.HasKeyword(core.UnblockableKW) {
		return true
	}
	if perm.HasKeyword(core.Flying) {
		return true
	}
	if perm.HasKeyword(core.Fear) {
		return true
	}
	if mage.HasLandwalkEvasion(perm, defenderID, g) {
		return true
	}
	return false
}

// estimatePushThroughDamage estimates how much damage attackerPlayerID can push
// through defenderPlayerID's blockers. Returns total damage and the list of
// attackers contributing.
func estimatePushThroughDamage(g *mage.Game, attackerPlayerID, defenderPlayerID uuid.UUID) (int, []uuid.UUID) {
	totalDmg := 0
	var attackerIDs []uuid.UUID

	// Collect defender's available blockers (untapped creatures)
	var blockers []*mage.Permanent
	for _, perm := range g.Battlefield {
		if perm.Controller == defenderPlayerID && perm.HasType(core.TypeCreature) && !perm.Tapped {
			if perm.CanDeclareAsBlocker(g) {
				blockers = append(blockers, perm)
			}
		}
	}

	// Find best blocker toughness (for trample estimation)
	bestBlockerToughness := 0
	for _, b := range blockers {
		t := b.CurrentToughness(g)
		if t > bestBlockerToughness {
			bestBlockerToughness = t
		}
	}

	// Count evasive damage first (bypasses blockers entirely)
	for _, perm := range g.Battlefield {
		if perm.Controller != attackerPlayerID || !perm.HasType(core.TypeCreature) {
			continue
		}
		if !perm.CanDeclareAsAttacker(g) {
			continue
		}
		pow := perm.CurrentPower(g)
		if pow <= 0 {
			continue
		}

		if isEvasive(perm, defenderPlayerID, g) {
			// Check if defender has flying blockers for flying attackers
			if perm.HasKeyword(core.Flying) && !perm.HasKeyword(core.UnblockableKW) {
				canBeBlocked := false
				for _, b := range blockers {
					if mage.CanBlock(b, perm, g) {
						canBeBlocked = true
						break
					}
				}
				if canBeBlocked {
					// Flying attacker can be blocked — treat as non-evasive with trample check
					if perm.HasKeyword(core.Trample) && pow > bestBlockerToughness {
						totalDmg += pow - bestBlockerToughness
						attackerIDs = append(attackerIDs, perm.ID())
					}
					continue
				}
			}
			totalDmg += pow
			attackerIDs = append(attackerIDs, perm.ID())
		} else if perm.HasKeyword(core.Trample) {
			// Trample: power minus best blocker toughness gets through
			if pow > bestBlockerToughness {
				totalDmg += pow - bestBlockerToughness
				attackerIDs = append(attackerIDs, perm.ID())
			}
		} else if perm.HasKeyword(core.Menace) {
			// Menace: needs two blockers — if fewer than 2 blockers available, gets through
			if len(blockers) < 2 {
				totalDmg += pow
				attackerIDs = append(attackerIDs, perm.ID())
			}
		}
	}

	return totalDmg, attackerIDs
}

// findMinimalLethalSet finds the smallest subset of evasive/trample attackers
// that deal at least targetLife damage. Prefers evasive creatures first, sorted
// by power descending, to minimize the number of attackers needed.
func findMinimalLethalSet(g *mage.Game, attackerPlayerID, defenderPlayerID uuid.UUID, targetLife int) []uuid.UUID {
	type attackerInfo struct {
		id  uuid.UUID
		dmg int
	}

	var candidates []attackerInfo

	// Collect defender blockers for trample calc
	bestBlockerToughness := 0
	for _, perm := range g.Battlefield {
		if perm.Controller == defenderPlayerID && perm.HasType(core.TypeCreature) && !perm.Tapped {
			if perm.CanDeclareAsBlocker(g) {
				t := perm.CurrentToughness(g)
				if t > bestBlockerToughness {
					bestBlockerToughness = t
				}
			}
		}
	}

	blockerCount := 0
	for _, perm := range g.Battlefield {
		if perm.Controller == defenderPlayerID && perm.HasType(core.TypeCreature) && !perm.Tapped {
			if perm.CanDeclareAsBlocker(g) {
				blockerCount++
			}
		}
	}

	for _, perm := range g.Battlefield {
		if perm.Controller != attackerPlayerID || !perm.HasType(core.TypeCreature) {
			continue
		}
		if !perm.CanDeclareAsAttacker(g) {
			continue
		}
		pow := perm.CurrentPower(g)
		if pow <= 0 {
			continue
		}

		dmg := 0
		if isEvasive(perm, defenderPlayerID, g) {
			// Check if flying can actually be blocked
			if perm.HasKeyword(core.Flying) && !perm.HasKeyword(core.UnblockableKW) {
				canBeBlocked := false
				for _, b := range g.Battlefield {
					if b.Controller == defenderPlayerID && b.HasType(core.TypeCreature) && !b.Tapped {
						if mage.CanBlock(b, perm, g) {
							canBeBlocked = true
							break
						}
					}
				}
				if canBeBlocked {
					if perm.HasKeyword(core.Trample) && pow > bestBlockerToughness {
						dmg = pow - bestBlockerToughness
					}
				} else {
					dmg = pow
				}
			} else {
				dmg = pow
			}
		} else if perm.HasKeyword(core.Trample) && pow > bestBlockerToughness {
			dmg = pow - bestBlockerToughness
		} else if perm.HasKeyword(core.Menace) && blockerCount < 2 {
			dmg = pow
		}

		if dmg > 0 {
			candidates = append(candidates, attackerInfo{id: perm.ID(), dmg: dmg})
		}
	}

	// Sort by damage descending (simple insertion sort for small N)
	for i := 1; i < len(candidates); i++ {
		for j := i; j > 0 && candidates[j].dmg > candidates[j-1].dmg; j-- {
			candidates[j], candidates[j-1] = candidates[j-1], candidates[j]
		}
	}

	// Greedily pick highest-damage attackers until we reach lethal
	var result []uuid.UUID
	remaining := targetLife
	for _, c := range candidates {
		if remaining <= 0 {
			break
		}
		result = append(result, c.id)
		remaining -= c.dmg
	}
	if remaining > 0 {
		return nil // can't reach lethal with evasive/trample alone
	}
	return result
}

// ─── Race Calculation (Phase 0B) ────────────────────────────────────────────

// RaceInfo describes the "clock" (turns to kill) for both players.
type RaceInfo struct {
	MyClock    int  // turns until I kill opponent (math.MaxInt32 if never)
	TheirClock int  // turns until opponent kills me
	Racing     bool // true if both clocks < 5
}

// CalculateRace computes turn clocks for both players. A clock of 1 means
// lethal next attack. Uses estimatePushThroughDamage for evasive/trample
// damage, plus estimated non-evasive damage that gets past blockers.
func CalculateRace(g *mage.Game, playerID uuid.UUID) RaceInfo {
	opponent := g.GetOpponent(playerID)
	if opponent == nil {
		return RaceInfo{MyClock: math.MaxInt32, TheirClock: math.MaxInt32}
	}
	oppID := opponent.PlayerID()
	me := g.GetPlayer(playerID)
	if me == nil {
		return RaceInfo{MyClock: math.MaxInt32, TheirClock: math.MaxInt32}
	}

	myExpectedDmg := estimateExpectedDamage(g, playerID, oppID)
	theirExpectedDmg := estimateExpectedDamage(g, oppID, playerID)

	myClock := clock(opponent.Life(), myExpectedDmg)
	theirClock := clock(me.Life(), theirExpectedDmg)

	return RaceInfo{
		MyClock:    myClock,
		TheirClock: theirClock,
		Racing:     myClock < 5 && theirClock < 5,
	}
}

// estimateExpectedDamage estimates total damage per attack for attackerPlayer
// against defenderPlayer, including both evasive damage and non-evasive damage
// that might get through (discounted by blocker availability).
func estimateExpectedDamage(g *mage.Game, attackerPlayerID, defenderPlayerID uuid.UUID) int {
	// Start with guaranteed push-through damage
	evasiveDmg, _ := estimatePushThroughDamage(g, attackerPlayerID, defenderPlayerID)

	// Add non-evasive damage that might get through, discounted
	nonEvasivePower := 0
	for _, perm := range g.Battlefield {
		if perm.Controller != attackerPlayerID || !perm.HasType(core.TypeCreature) {
			continue
		}
		if !perm.CanDeclareAsAttacker(g) {
			continue
		}
		pow := perm.CurrentPower(g)
		if pow <= 0 {
			continue
		}
		// Skip already-counted evasive and trample creatures
		if isEvasive(perm, defenderPlayerID, g) || perm.HasKeyword(core.Trample) || perm.HasKeyword(core.Menace) {
			continue
		}
		nonEvasivePower += pow
	}

	// Count available blockers
	blockerCount := 0
	blockerToughness := 0
	for _, perm := range g.Battlefield {
		if perm.Controller == defenderPlayerID && perm.HasType(core.TypeCreature) && !perm.Tapped {
			if perm.CanDeclareAsBlocker(g) {
				blockerCount++
				blockerToughness += perm.CurrentToughness(g)
			}
		}
	}

	// Estimate non-evasive damage: assume blockers absorb some.
	// If we have more attackers than they have blockers, extras get through.
	nonEvasiveCount := 0
	for _, perm := range g.Battlefield {
		if perm.Controller == attackerPlayerID && perm.HasType(core.TypeCreature) {
			if !perm.CanDeclareAsAttacker(g) {
				continue
			}
			if perm.CurrentPower(g) <= 0 {
				continue
			}
			if !isEvasive(perm, defenderPlayerID, g) && !perm.HasKeyword(core.Trample) && !perm.HasKeyword(core.Menace) {
				nonEvasiveCount++
			}
		}
	}

	unblocked := nonEvasiveCount - blockerCount
	if unblocked < 0 {
		unblocked = 0
	}
	// Estimate: unblocked creatures' average power gets through
	nonEvasiveDmg := 0
	if nonEvasiveCount > 0 && unblocked > 0 {
		avgPow := nonEvasivePower / nonEvasiveCount
		nonEvasiveDmg = avgPow * unblocked
	}

	return evasiveDmg + nonEvasiveDmg
}

// clock returns ceil(life / damagePerTurn). Returns math.MaxInt32 if damage is 0.
func clock(life, damagePerTurn int) int {
	if damagePerTurn <= 0 {
		return math.MaxInt32
	}
	return (life + damagePerTurn - 1) / damagePerTurn
}

// ─── Mana Curve Awareness (Phase 0C) ────────────────────────────────────────

// manaCurveBonus scores a spell based on how well it uses available mana.
// It rewards casting spells that use most of the available mana when
// higher-CMC options exist in hand.
func manaCurveBonus(cardCMC, availableMana int, handCMCs []int) float64 {
	if availableMana <= 0 {
		return 0
	}

	// Find the max CMC we could cast from hand
	maxCastable := 0
	for _, cmc := range handCMCs {
		if cmc <= availableMana && cmc > maxCastable {
			maxCastable = cmc
		}
	}

	// If this is the most expensive castable spell, give it a bonus
	if cardCMC == maxCastable {
		return 3
	}

	// Penalize low-CMC spells when higher-CMC options are castable
	if maxCastable > 0 && cardCMC < maxCastable {
		wastedMana := availableMana - cardCMC
		// Only penalize if we're leaving a lot of mana unused AND could use it
		if wastedMana >= 2 {
			return -2
		}
	}

	return 0
}

// countAvailableMana counts the number of untapped mana sources a player controls.
func countAvailableMana(g *mage.Game, playerID uuid.UUID) int {
	count := 0
	for _, perm := range g.Battlefield {
		if perm.Controller != playerID {
			continue
		}
		if perm.Tapped {
			continue
		}
		// Count lands and mana-producing artifacts/creatures
		if perm.HasType(core.TypeLand) {
			count++
			continue
		}
		// Check for mana abilities
		for _, a := range perm.RuntimeAbilities {
			if _, ok := mage.UnwrapAbility(a).(*mage.ManaAbility); ok {
				count++
				break
			}
		}
	}
	return count
}

// handCMCs returns the CMC of each non-land, non-instant card in a player's hand.
func handCMCs(hand []mage.Card) []int {
	var cmcs []int
	for _, c := range hand {
		if c.HasType(core.TypeLand) || c.HasType(core.TypeInstant) {
			continue
		}
		cmcs = append(cmcs, c.ManaCost().CMC())
	}
	return cmcs
}
