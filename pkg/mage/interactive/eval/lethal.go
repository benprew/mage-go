package eval

import (
	"math"

	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// LethalInfo describes whether either player can push through lethal damage.
type LethalInfo struct {
	IHaveLethal      bool
	TheyHaveLethal   bool
	MyBoardDamage    int
	TheirBoardDamage int
	LethalAttackers  []uuid.UUID
}

// CalculateLethal computes lethal-on-board information for playerID.
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

// IsEvasive returns true if the attacker has evasion that lets it bypass most blockers.
func IsEvasive(perm *mage.Permanent, defenderID uuid.UUID, g *mage.Game) bool {
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

func estimatePushThroughDamage(g *mage.Game, attackerPlayerID, defenderPlayerID uuid.UUID) (int, []uuid.UUID) {
	totalDmg := 0
	var attackerIDs []uuid.UUID

	var blockers []*mage.Permanent
	for _, perm := range g.Battlefield {
		if perm.Controller == defenderPlayerID && perm.HasType(core.TypeCreature) && !perm.Tapped {
			if perm.CanDeclareAsBlocker(g) {
				blockers = append(blockers, perm)
			}
		}
	}

	bestBlockerToughness := 0
	for _, b := range blockers {
		t := b.CurrentToughness(g)
		if t > bestBlockerToughness {
			bestBlockerToughness = t
		}
	}

	// Effective blocker toughness for trample/push-through calculations.
	// A deathtouch attacker only needs 1 damage to kill any blocker.
	effectiveBlockerTough := func(perm *mage.Permanent) int {
		if perm.HasKeyword(core.Deathtouch) {
			return 1
		}
		return bestBlockerToughness
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

		hasDT := perm.HasKeyword(core.Deathtouch)
		hasFS := perm.HasKeyword(core.FirstStrike) || perm.HasKeyword(core.DoubleStrike)
		effBlockTough := effectiveBlockerTough(perm)

		if IsEvasive(perm, defenderPlayerID, g) {
			if perm.HasKeyword(core.Flying) && !perm.HasKeyword(core.UnblockableKW) {
				canBeBlocked := false
				for _, b := range blockers {
					if mage.CanBlock(b, perm, g) {
						canBeBlocked = true
						break
					}
				}
				if canBeBlocked {
					if perm.HasKeyword(core.Trample) && pow > effBlockTough {
						totalDmg += pow - effBlockTough
						attackerIDs = append(attackerIDs, perm.ID())
					} else if hasFS && pow >= bestBlockerToughness {
						// First strike kills the blocker before it can deal damage.
						// All damage effectively pushes through.
						totalDmg += pow
						attackerIDs = append(attackerIDs, perm.ID())
					} else if hasDT && hasFS {
						// Deathtouch first striker kills any blocker before damage.
						totalDmg += pow
						attackerIDs = append(attackerIDs, perm.ID())
					}
					continue
				}
			}
			totalDmg += pow
			attackerIDs = append(attackerIDs, perm.ID())
		} else if perm.HasKeyword(core.Trample) {
			if pow > effBlockTough {
				totalDmg += pow - effBlockTough
				attackerIDs = append(attackerIDs, perm.ID())
			}
		} else if hasFS && len(blockers) > 0 && pow >= bestBlockerToughness {
			// First striker that kills the best blocker before taking damage.
			// Conservatively estimate full power as push-through damage since
			// the blocker dies before dealing damage back.
			totalDmg += pow
			attackerIDs = append(attackerIDs, perm.ID())
		} else if hasDT && hasFS && len(blockers) > 0 {
			// Deathtouch + first strike: kills any blocker before damage.
			totalDmg += pow
			attackerIDs = append(attackerIDs, perm.ID())
		} else if perm.HasKeyword(core.Menace) {
			if len(blockers) < 2 {
				totalDmg += pow
				attackerIDs = append(attackerIDs, perm.ID())
			}
		}
	}

	return totalDmg, attackerIDs
}

func findMinimalLethalSet(g *mage.Game, attackerPlayerID, defenderPlayerID uuid.UUID, targetLife int) []uuid.UUID {
	type attackerInfo struct {
		id  uuid.UUID
		dmg int
	}

	var candidates []attackerInfo

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

		hasDT := perm.HasKeyword(core.Deathtouch)
		hasFS := perm.HasKeyword(core.FirstStrike) || perm.HasKeyword(core.DoubleStrike)
		effBlockTough := bestBlockerToughness
		if hasDT {
			effBlockTough = 1
		}

		dmg := 0
		if IsEvasive(perm, defenderPlayerID, g) {
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
					if perm.HasKeyword(core.Trample) && pow > effBlockTough {
						dmg = pow - effBlockTough
					} else if hasFS && pow >= bestBlockerToughness {
						dmg = pow
					} else if hasDT && hasFS {
						dmg = pow
					}
				} else {
					dmg = pow
				}
			} else {
				dmg = pow
			}
		} else if perm.HasKeyword(core.Trample) && pow > effBlockTough {
			dmg = pow - effBlockTough
		} else if hasFS && blockerCount > 0 && pow >= bestBlockerToughness {
			dmg = pow
		} else if hasDT && hasFS && blockerCount > 0 {
			dmg = pow
		} else if perm.HasKeyword(core.Menace) && blockerCount < 2 {
			dmg = pow
		}

		if dmg > 0 {
			candidates = append(candidates, attackerInfo{id: perm.ID(), dmg: dmg})
		}
	}

	for i := 1; i < len(candidates); i++ {
		for j := i; j > 0 && candidates[j].dmg > candidates[j-1].dmg; j-- {
			candidates[j], candidates[j-1] = candidates[j-1], candidates[j]
		}
	}

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
		return nil
	}
	return result
}

// RaceInfo describes the "clock" (turns to kill) for both players.
type RaceInfo struct {
	MyClock    int
	TheirClock int
	Racing     bool
}

// CalculateRace computes turn clocks for both players.
// Lifelink is accounted for: a player with lifelink attackers gains life each turn,
// making the opponent's clock longer.
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

	// Estimate lifelink life gain per turn for each player.
	myLifelinkGain := estimateLifelinkGain(g, playerID)
	theirLifelinkGain := estimateLifelinkGain(g, oppID)

	// My clock: turns for me to kill opponent. Opponent's lifelink gain extends this.
	myClock := clock(opponent.Life(), myExpectedDmg-theirLifelinkGain)
	// Their clock: turns for them to kill me. My lifelink gain extends this.
	theirClock := clock(me.Life(), theirExpectedDmg-myLifelinkGain)

	return RaceInfo{
		MyClock:    myClock,
		TheirClock: theirClock,
		Racing:     myClock < 5 && theirClock < 5,
	}
}

// estimateLifelinkGain returns the estimated life gain per turn from a player's
// lifelink attackers. Counts evasive lifelink at full value, and non-evasive
// lifelink at partial value (they connect some of the time via combat).
func estimateLifelinkGain(g *mage.Game, playerID uuid.UUID) int {
	opponent := g.GetOpponent(playerID)
	if opponent == nil {
		return 0
	}
	oppID := opponent.PlayerID()

	evasiveGain := 0
	nonEvasiveGain := 0
	for _, perm := range g.Battlefield {
		if perm.Controller != playerID || !perm.HasType(core.TypeCreature) {
			continue
		}
		if !perm.HasKeyword(core.Lifelink) {
			continue
		}
		if !perm.CanDeclareAsAttacker(g) {
			continue
		}
		pow := perm.CurrentPower(g)
		if pow <= 0 {
			continue
		}
		if IsEvasive(perm, oppID, g) {
			evasiveGain += pow
		} else {
			nonEvasiveGain += pow
		}
	}

	// Non-evasive lifelink creatures still gain life when they connect in combat
	// (either unblocked or dealing damage while blocked). Estimate ~50% connect rate.
	return evasiveGain + nonEvasiveGain/2
}

func estimateExpectedDamage(g *mage.Game, attackerPlayerID, defenderPlayerID uuid.UUID) int {
	evasiveDmg, _ := estimatePushThroughDamage(g, attackerPlayerID, defenderPlayerID)

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
		if IsEvasive(perm, defenderPlayerID, g) || perm.HasKeyword(core.Trample) || perm.HasKeyword(core.Menace) {
			continue
		}
		nonEvasivePower += pow
	}

	blockerCount := 0
	for _, perm := range g.Battlefield {
		if perm.Controller == defenderPlayerID && perm.HasType(core.TypeCreature) && !perm.Tapped {
			if perm.CanDeclareAsBlocker(g) {
				blockerCount++
			}
		}
	}

	nonEvasiveCount := 0
	for _, perm := range g.Battlefield {
		if perm.Controller == attackerPlayerID && perm.HasType(core.TypeCreature) {
			if !perm.CanDeclareAsAttacker(g) {
				continue
			}
			if perm.CurrentPower(g) <= 0 {
				continue
			}
			if !IsEvasive(perm, defenderPlayerID, g) && !perm.HasKeyword(core.Trample) && !perm.HasKeyword(core.Menace) {
				nonEvasiveCount++
			}
		}
	}

	unblocked := nonEvasiveCount - blockerCount
	if unblocked < 0 {
		unblocked = 0
	}
	nonEvasiveDmg := 0
	if nonEvasiveCount > 0 && unblocked > 0 {
		// Sort non-evasive by power descending and sum the top `unblocked` powers.
		// This is more accurate than average because the strongest creatures
		// are most likely to be blocked, meaning smaller ones sneak through.
		// But we take a conservative estimate: the weakest unblocked ones connect.
		avgPow := nonEvasivePower / nonEvasiveCount
		nonEvasiveDmg = avgPow * unblocked
	}
	// Even when all non-evasive creatures are blocked, creatures with first strike
	// or high power that survive combat still contribute to the clock by killing
	// blockers and opening up future attacks. Add a small bonus.
	if nonEvasiveCount > 0 && unblocked == 0 && blockerCount > 0 {
		// Estimate that attacks trade blockers over time, gradually opening the board.
		// Credit 1 point of damage per 2 non-evasive attackers as attrition bonus.
		nonEvasiveDmg = nonEvasiveCount / 2
	}

	return evasiveDmg + nonEvasiveDmg
}

func clock(life, damagePerTurn int) int {
	if damagePerTurn <= 0 {
		return math.MaxInt32
	}
	return (life + damagePerTurn - 1) / damagePerTurn
}
