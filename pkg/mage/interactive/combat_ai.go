package interactive

import (
	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// ─── Combat Outcome Evaluation (Phase 5D) ───────────────────────────────────

// CombatScore summarizes the outcome of a combat step.
type CombatScore struct {
	DamageToOpponent   int
	OurCreaturesLost   int
	TheirCreaturesLost int
	Score              int // net advantage: positive = good for us
}

// evaluateCombatOutcome estimates the result of an attack where `attackers` attack
// and `blocks` describes the defender's blocking assignments. Each attacker without
// a block assignment deals damage to the opponent. For blocked attackers, we simulate
// damage exchange.
func evaluateCombatOutcome(g *mage.Game, playerID uuid.UUID, attackers []uuid.UUID, blocks []mage.BlockAssignment) CombatScore {
	opponent := g.GetOpponent(playerID)
	if opponent == nil {
		return CombatScore{}
	}
	oppID := opponent.PlayerID()

	// Build a map of attacker -> blockers
	blockerMap := make(map[uuid.UUID][]uuid.UUID)
	for _, b := range blocks {
		blockerMap[b.AttackerID] = append(blockerMap[b.AttackerID], b.BlockerID)
	}

	var cs CombatScore

	for _, atkID := range attackers {
		atk := g.FindPermanent(atkID)
		if atk == nil {
			continue
		}
		atkPow := atk.CurrentPower(g)
		atkTough := atk.CurrentToughness(g)

		blockerIDs, isBlocked := blockerMap[atkID]
		if !isBlocked || len(blockerIDs) == 0 {
			// Unblocked: damage to opponent
			if atkPow > 0 {
				cs.DamageToOpponent += atkPow
			}
			continue
		}

		// Blocked: simulate damage exchange
		remainingAtkDmg := atkPow
		totalBlockerDmg := 0

		for _, blkID := range blockerIDs {
			blk := g.FindPermanent(blkID)
			if blk == nil {
				continue
			}
			blkPow := blk.CurrentPower(g)
			blkTough := blk.CurrentToughness(g)

			totalBlockerDmg += blkPow

			// Attacker assigns lethal to this blocker
			if remainingAtkDmg >= blkTough {
				cs.TheirCreaturesLost++
				remainingAtkDmg -= blkTough
			} else {
				remainingAtkDmg = 0
			}
		}

		// Trample overflow
		if remainingAtkDmg > 0 && atk.HasKeyword(core.Trample) {
			cs.DamageToOpponent += remainingAtkDmg
		}

		// Does attacker die?
		if totalBlockerDmg >= atkTough {
			cs.OurCreaturesLost++
		}
	}

	// Score: creature advantage (their losses - our losses) * creature value weight,
	// plus damage dealt
	cs.Score = cs.DamageToOpponent +
		cs.TheirCreaturesLost*6 -
		cs.OurCreaturesLost*6

	// Refine score by actual evalCreature values
	var ourLostValue, theirLostValue int
	for _, atkID := range attackers {
		atk := g.FindPermanent(atkID)
		if atk == nil {
			continue
		}
		blockerIDs := blockerMap[atkID]
		if len(blockerIDs) == 0 {
			continue
		}
		atkPow := atk.CurrentPower(g)
		atkTough := atk.CurrentToughness(g)
		totalBlockerDmg := 0
		remainingDmg := atkPow
		for _, blkID := range blockerIDs {
			blk := g.FindPermanent(blkID)
			if blk == nil {
				continue
			}
			totalBlockerDmg += blk.CurrentPower(g)
			blkTough := blk.CurrentToughness(g)
			if remainingDmg >= blkTough {
				theirLostValue += evalCreature(blk)
				remainingDmg -= blkTough
			}
		}
		if totalBlockerDmg >= atkTough {
			ourLostValue += evalCreature(atk)
		}
	}

	// Adjust score with actual creature values
	cs.Score = cs.DamageToOpponent + theirLostValue - ourLostValue
	_ = oppID

	return cs
}

// ─── Gang Block Detection (Phase 5C) ────────────────────────────────────────

// findGangBlocks finds a pair of available blockers whose combined power is
// enough to kill the attacker. Returns nil if no viable gang block exists, or
// if the trade is not worthwhile (attacker value <= sum of blocker values).
func findGangBlocks(atk *mage.Permanent, available []*mage.Permanent, g *mage.Game, playerID uuid.UUID, theyHaveLethal bool) []*mage.Permanent {
	atkTough := atk.CurrentToughness(g)
	atkScore := evalCreature(atk)

	// Try all pairs of available blockers
	for i := 0; i < len(available); i++ {
		if available[i] == nil {
			continue
		}
		b1 := available[i]
		if !mage.CanBlock(b1, atk, g) {
			continue
		}

		for j := i + 1; j < len(available); j++ {
			if available[j] == nil {
				continue
			}
			b2 := available[j]
			if !mage.CanBlock(b2, atk, g) {
				continue
			}

			combinedPow := b1.CurrentPower(g) + b2.CurrentPower(g)
			if combinedPow < atkTough {
				continue
			}

			// Check if the trade is worthwhile: attacker score >= sum of blocker scores.
			// Trading two small creatures for one big threat of equal value is
			// worthwhile because removing a single large threat is strategically
			// favorable. When facing lethal, gang-block regardless.
			b1Score := evalCreature(b1)
			b2Score := evalCreature(b2)
			if !theyHaveLethal && atkScore < b1Score+b2Score {
				continue
			}

			return []*mage.Permanent{b1, b2}
		}
	}

	return nil
}

// canSingleBlockKill returns true if any single available blocker can kill the attacker.
func canSingleBlockKill(atk *mage.Permanent, available []*mage.Permanent, g *mage.Game) bool {
	atkTough := atk.CurrentToughness(g)
	for _, blk := range available {
		if blk == nil {
			continue
		}
		if !mage.CanBlock(blk, atk, g) {
			continue
		}
		if blk.CurrentPower(g) >= atkTough {
			return true
		}
	}
	return false
}

// ─── Hold-Back Heuristic (Phase 5B) ─────────────────────────────────────────

// holdBackValue calculates the expected value of holding mana open for instants
// versus spending it on the best sorcery-speed play. Returns a value >= 0.
// Higher value means stronger reason to hold mana open.
func holdBackValue(p mage.Player, g *mage.Game, w WeightedPersonality) float64 {
	playerID := p.PlayerID()

	// Find best instant value in hand
	bestInstantValue := 0.0
	for _, card := range p.Hand() {
		if !card.HasType(core.TypeInstant) {
			continue
		}
		if !g.CanAfford(playerID, card.ManaCost()) {
			continue
		}
		// Check it has usable effects
		hasUsableEffect := false
		for _, a := range card.Abilities() {
			if sa, ok := a.(*mage.SpellAbility); ok {
				if mage.SpellOutcome(sa.Effects()) != mage.OutcomeUnknown {
					hasUsableEffect = true
					break
				}
			}
		}
		if !hasUsableEffect {
			continue
		}
		sv := float64(spellValue(card, p, g))
		// Scale instant value by potential response benefit:
		// - Removal is much better as an instant (can respond to threats)
		// - Pump spells are great as combat tricks
		for _, a := range card.Abilities() {
			if sa, ok := a.(*mage.SpellAbility); ok {
				outcome := mage.SpellOutcome(sa.Effects())
				if outcome == mage.OutcomeDetriment {
					sv *= 1.5 // removal is better saved for response
				}
				if outcome == mage.OutcomeBenefit {
					sv *= 1.3 // pump/protection is better as combat trick
				}
			}
		}
		if sv > bestInstantValue {
			bestInstantValue = sv
		}
	}

	// Find best sorcery-speed play value
	bestSorceryValue := 0.0
	for _, card := range g.GetCastableSpells(playerID) {
		if card.HasType(core.TypeInstant) {
			continue
		}
		sv := float64(spellValue(card, p, g))
		if sv > bestSorceryValue {
			bestSorceryValue = sv
		}
	}

	// Hold-back if instant value * HoldInstants weight > sorcery value
	// At HoldInstants=0: never hold (threshold is infinite)
	// At HoldInstants=0.5: hold if instant value > 2x sorcery value
	// At HoldInstants=1.0: hold if instant value > sorcery value
	if w.HoldInstants <= 0 {
		return 0
	}

	// Scale: at HoldInstants=1.0, hold if instant >= sorcery
	// At HoldInstants=0.5, hold if instant >= 2*sorcery
	threshold := bestSorceryValue / w.HoldInstants
	if bestInstantValue >= threshold && bestInstantValue > 0 {
		return bestInstantValue - threshold
	}
	return 0
}

// ─── Response Evaluation (Phase 5A) ─────────────────────────────────────────

// evaluateResponse checks if we have instants that could usefully respond
// when it's not our main phase (opponent's turn, or in response to something).
// Returns a PriorityAction to cast an instant, or nil to pass.
func (s *HeuristicStrategy) evaluateResponse(p mage.Player, g *mage.Game) *PriorityAction {
	playerID := p.PlayerID()
	opponent := g.GetOpponent(playerID)
	if opponent == nil {
		return nil
	}

	// Check if there's something on the stack we should respond to
	stackHasThreat := false
	if len(g.Stack.Objects()) > 0 {
		for _, obj := range g.Stack.Objects() {
			if obj.Controller != playerID {
				stackHasThreat = true
				break
			}
		}
	}

	// Evaluate each instant in hand for response value
	var bestAction *PriorityAction
	bestValue := 0

	for _, card := range p.Hand() {
		if !card.HasType(core.TypeInstant) {
			continue
		}
		if !g.CanAfford(playerID, card.ManaCost()) {
			continue
		}

		// Check it has usable effects
		hasUsableEffect := false
		for _, a := range card.Abilities() {
			if sa, ok := a.(*mage.SpellAbility); ok {
				if mage.SpellOutcome(sa.Effects()) != mage.OutcomeUnknown {
					hasUsableEffect = true
					break
				}
			}
		}
		if !hasUsableEffect {
			continue
		}

		sv := spellValue(card, p, g)

		// Boost value if there's an opponent spell on the stack
		// (more likely our response matters)
		if stackHasThreat {
			sv += 3
		}

		// Check if this is removal and opponent has valuable creatures
		for _, a := range card.Abilities() {
			sa, ok := a.(*mage.SpellAbility)
			if !ok {
				continue
			}
			outcome := mage.SpellOutcome(sa.Effects())
			if outcome == mage.OutcomeDetriment {
				// Removal: high value as response
				sv += 2
			}
		}

		if sv > bestValue {
			targets := s.autoSelectTargets(p, g, card)
			if len(targets) > 0 {
				bestValue = sv
				bestAction = &PriorityAction{
					Type:     ActionCastSpell,
					CardID:   card.ID(),
					CardName: card.Name(),
					Targets:  targets,
				}
			}
		}
	}

	// Only respond if the value is high enough to be worth it
	// (don't waste instants on marginal responses)
	if bestAction != nil && bestValue >= 3 {
		return bestAction
	}

	return nil
}

// ─── Race-Informed Combat Helpers (Phase 5E) ────────────────────────────────

// raceInformedAttack refines the attacker list based on race state.
// When racing favorably: attack with everything (already handled in Attackers).
// When racing unfavorably: only attack with evasion creatures.
// When tied: only attack with creatures that trade up or have evasion.
func raceInformedAttack(perm *mage.Permanent, g *mage.Game, opponentID uuid.UUID, race RaceInfo) bool {
	if !race.Racing {
		return true // not racing, no race filter
	}

	if race.MyClock < race.TheirClock {
		return true // racing favorably, attack with everything
	}

	if race.TheirClock < race.MyClock {
		// Racing unfavorably: only attack with evasion
		return isEvasive(perm, opponentID, g)
	}

	// Tied: attack with evasion or profitable trades
	if isEvasive(perm, opponentID, g) {
		return true
	}
	return profitableToAttack(perm, g, opponentID)
}

// raceInformedBlock decides if we should block this attacker given the race.
// When racing favorably: only chump-block if it saves a clock turn
// When racing unfavorably: block everything aggressively
// When tied: trade up when possible
func raceInformedBlock(atk *mage.Permanent, blk *mage.Permanent, g *mage.Game, race RaceInfo) bool {
	if !race.Racing {
		return true // not racing, allow block
	}

	atkPow := atk.CurrentPower(g)

	if race.MyClock < race.TheirClock {
		// Racing favorably: skip blocking small stuff
		// Only block if the damage would be significant (> 25% of our life)
		me := g.GetPlayer(blk.Controller)
		if me != nil && atkPow*4 < me.Life() {
			return false // small attacker, ignore it
		}
		return true
	}

	if race.TheirClock < race.MyClock {
		// Racing unfavorably: block aggressively, trade up
		blkPow := blk.CurrentPower(g)
		atkTough := atk.CurrentToughness(g)
		if blkPow >= atkTough {
			return true // can kill attacker
		}
		// Still block to absorb damage even if we can't kill it
		return true
	}

	// Tied: trade up when possible
	blkPow := blk.CurrentPower(g)
	atkTough := atk.CurrentToughness(g)
	blkTough := blk.CurrentToughness(g)
	if blkPow >= atkTough {
		return true // we kill attacker
	}
	if atkPow < blkTough {
		return true // we survive the block
	}
	// Would be a bad trade, skip
	return false
}
