package eval

import (
	"math"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
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

// controllerCanGrantEvasionTo returns the evasion keyword the controller could
// grant to perm via an activatable ability, or 0 if none.
//
// Limitations: this is a heuristic for the lethal short-circuit. It assumes any
// "TargetCreature" granter can reach perm and that activation cost is payable
// in isolation. Self-targeting grants (KindSource) only count when the source
// is perm itself.
func controllerCanGrantEvasionTo(perm *mage.Permanent, g *mage.Game) core.Attr {
	controller := perm.ControllerID()
	for _, src := range g.AllBattlefield() {
		if src.ControllerID() != controller {
			continue
		}
		for _, ab := range src.RuntimeAbilities {
			act, ok := mage.UnwrapAbility(ab).(mage.ActivatedAbility)
			if !ok {
				continue
			}
			if !act.CanActivate(controller, g) {
				continue
			}
			for _, e := range act.Effects() {
				kw := e.Properties().GrantedKeyword
				if kw == 0 || !isEvasionGrantUseful(kw) {
					continue
				}
				if len(act.Targets()) == 0 && src.ID() != perm.ID() {
					continue
				}
				return kw
			}
		}
	}
	return 0
}

func isEvasionGrantUseful(kw core.Attr) bool {
	return kw == core.Flying || kw == core.UnblockableKW
}

// prospectiveUnblockable returns true if perm is not currently evasive but
// could be granted an evasion keyword that no blocker can answer.
func prospectiveUnblockable(perm *mage.Permanent, blockers []*mage.Permanent, g *mage.Game) bool {
	kw := controllerCanGrantEvasionTo(perm, g)
	if kw == 0 {
		return false
	}
	for _, b := range blockers {
		switch kw {
		case core.UnblockableKW:
			return false
		case core.Flying:
			if b.HasKeyword(core.Flying) || b.HasKeyword(core.Reach) {
				return false
			}
		}
	}
	return true
}

// isProspectivelyUnblockable is the same check as prospectiveUnblockable but
// computes the defender's blockers from the game state.
func isProspectivelyUnblockable(perm *mage.Permanent, defenderID uuid.UUID, g *mage.Game) bool {
	var blockers []*mage.Permanent
	for _, b := range g.AllBattlefield() {
		if b.ControllerID() == defenderID && b.HasType(core.TypeCreature) && !b.Tapped && b.CanDeclareAsBlocker(g) {
			blockers = append(blockers, b)
		}
	}
	return prospectiveUnblockable(perm, blockers, g)
}

func estimatePushThroughDamage(g *mage.Game, attackerPlayerID, defenderPlayerID uuid.UUID) (int, []uuid.UUID) {
	totalDmg := 0
	var attackerIDs []uuid.UUID

	var blockers []*mage.Permanent
	for _, perm := range g.AllBattlefield() {
		if perm.ControllerID() == defenderPlayerID && perm.HasType(core.TypeCreature) && !perm.Tapped {
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

	for _, perm := range g.AllBattlefield() {
		if perm.ControllerID() != attackerPlayerID || !perm.HasType(core.TypeCreature) {
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
					effPow := pow + maxAffordablePumpBoost(perm, g)
					if perm.HasKeyword(core.Trample) && effPow > effBlockTough {
						totalDmg += effPow - effBlockTough
						attackerIDs = append(attackerIDs, perm.ID())
					} else if hasFS && effPow >= bestBlockerToughness {
						// First strike kills the blocker before it can deal damage.
						// All damage effectively pushes through.
						totalDmg += effPow
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
		} else if prospectiveUnblockable(perm, blockers, g) {
			// Controller has an activatable ability (e.g. Flying Carpet) that
			// would make this attacker unblockable against the current defender
			// board. Treat as evasive for lethal purposes.
			totalDmg += pow
			attackerIDs = append(attackerIDs, perm.ID())
		} else if perm.HasKeyword(core.Trample) {
			effPow := pow + maxAffordablePumpBoost(perm, g)
			if effPow > effBlockTough {
				totalDmg += effPow - effBlockTough
				attackerIDs = append(attackerIDs, perm.ID())
			}
		} else if hasFS && len(blockers) > 0 && pow+maxAffordablePumpBoost(perm, g) >= bestBlockerToughness {
			// First striker that kills the best blocker before taking damage,
			// pumping its power over the threshold first if needed and
			// affordable. Conservatively estimate full (pumped) power as
			// push-through damage since the blocker dies before dealing
			// damage back.
			totalDmg += pow + maxAffordablePumpBoost(perm, g)
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
	for _, perm := range g.AllBattlefield() {
		if perm.ControllerID() == defenderPlayerID && perm.HasType(core.TypeCreature) && !perm.Tapped {
			if perm.CanDeclareAsBlocker(g) {
				t := perm.CurrentToughness(g)
				if t > bestBlockerToughness {
					bestBlockerToughness = t
				}
			}
		}
	}

	blockerCount := 0
	for _, perm := range g.AllBattlefield() {
		if perm.ControllerID() == defenderPlayerID && perm.HasType(core.TypeCreature) && !perm.Tapped {
			if perm.CanDeclareAsBlocker(g) {
				blockerCount++
			}
		}
	}

	for _, perm := range g.AllBattlefield() {
		if perm.ControllerID() != attackerPlayerID || !perm.HasType(core.TypeCreature) {
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
				for _, b := range g.AllBattlefield() {
					if b.ControllerID() == defenderPlayerID && b.HasType(core.TypeCreature) && !b.Tapped {
						if mage.CanBlock(b, perm, g) {
							canBeBlocked = true
							break
						}
					}
				}
				if canBeBlocked {
					effPow := pow + maxAffordablePumpBoost(perm, g)
					if perm.HasKeyword(core.Trample) && effPow > effBlockTough {
						dmg = effPow - effBlockTough
					} else if hasFS && effPow >= bestBlockerToughness {
						dmg = effPow
					} else if hasDT && hasFS {
						dmg = pow
					}
				} else {
					dmg = pow
				}
			} else {
				dmg = pow
			}
		} else if isProspectivelyUnblockable(perm, defenderPlayerID, g) {
			dmg = pow
		} else if perm.HasKeyword(core.Trample) && pow+maxAffordablePumpBoost(perm, g) > effBlockTough {
			dmg = pow + maxAffordablePumpBoost(perm, g) - effBlockTough
		} else if hasFS && blockerCount > 0 && pow+maxAffordablePumpBoost(perm, g) >= bestBlockerToughness {
			dmg = pow + maxAffordablePumpBoost(perm, g)
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

// maxAffordablePumpBoost returns the total power perm's controller could add
// to it this combat via self-targeting activated pump abilities they can
// currently afford, assuming (as the other lethal-calc heuristics in this
// file do, e.g. controllerCanGrantEvasionTo) that the mana is payable in
// isolation from anything else the controller might want to do. Used so
// push-through estimates account for a creature that can pump over a
// blocker's toughness (trample) or up to it (killing the blocker with first
// strike before damage), not just its current power.
func maxAffordablePumpBoost(perm *mage.Permanent, g *mage.Game) int {
	total := 0
	for _, ab := range perm.RuntimeAbilities {
		act, ok := mage.UnwrapAbility(ab).(mage.ActivatedAbility)
		if !ok || len(act.Targets()) != 0 {
			continue
		}
		boost := 0
		for _, e := range act.Effects() {
			boost += e.Properties().PowerBoost
		}
		if boost <= 0 || !act.CanActivate(perm.ControllerID(), g) {
			continue
		}
		total += boost * maxRepeatablePumpActivations(act, g, perm.ControllerID())
	}
	return total
}

// maxRepeatablePumpActivations returns how many times the controller can pay
// for ab from currently available mana. Abilities with a non-mana cost (tap,
// sacrifice) or an {X} cost can be used at most once, since those costs can't
// be repaid repeatedly this priority pass.
func maxRepeatablePumpActivations(ab mage.ActivatedAbility, g *mage.Game, controller uuid.UUID) int {
	per := core.ManaCost{}
	repeatable := true
	for _, c := range ab.Costs() {
		mp, ok := c.(*mage.ManaCostPayment)
		if !ok || mp.MC.HasX {
			repeatable = false
			continue
		}
		per = addManaCost(per, mp.MC)
	}
	if !repeatable {
		return 1
	}
	combined := core.ManaCost{}
	count := 0
	const maxActivations = 20
	for count < maxActivations {
		combined = addManaCost(combined, per)
		if !g.CanAfford(controller, combined, nil) {
			break
		}
		count++
	}
	return count
}

// addManaCost returns the sum of two mana costs (ignoring {X}, handled by callers).
func addManaCost(a, b core.ManaCost) core.ManaCost {
	a.Generic += b.Generic
	a.White += b.White
	a.Blue += b.Blue
	a.Black += b.Black
	a.Red += b.Red
	a.Green += b.Green
	a.Hybrid = append(append([]core.HybridSymbol{}, a.Hybrid...), b.Hybrid...)
	return a
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
	for _, perm := range g.AllBattlefield() {
		if perm.ControllerID() != playerID || !perm.HasType(core.TypeCreature) {
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
	for _, perm := range g.AllBattlefield() {
		if perm.ControllerID() != attackerPlayerID || !perm.HasType(core.TypeCreature) {
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
	for _, perm := range g.AllBattlefield() {
		if perm.ControllerID() == defenderPlayerID && perm.HasType(core.TypeCreature) && !perm.Tapped {
			if perm.CanDeclareAsBlocker(g) {
				blockerCount++
			}
		}
	}

	nonEvasiveCount := 0
	for _, perm := range g.AllBattlefield() {
		if perm.ControllerID() == attackerPlayerID && perm.HasType(core.TypeCreature) {
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

	unblocked := max(nonEvasiveCount-blockerCount, 0)
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
