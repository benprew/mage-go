package ai

import (
	"github.com/google/uuid"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/eval"
)

// CombatScore summarizes the outcome of a combat step.
type CombatScore struct {
	DamageToOpponent   int
	OurCreaturesLost   int
	TheirCreaturesLost int
	LifeGained         int // life gained by our lifelink attackers
	OpponentLifeGained int // life gained by opponent's lifelink blockers
	Score              int
}

func evaluateCombatOutcome(g *mage.Game, playerID uuid.UUID, attackers []uuid.UUID, blocks []mage.BlockAssignment) CombatScore {
	opponent := g.GetOpponent(playerID)
	if opponent == nil {
		return CombatScore{}
	}
	oppID := opponent.PlayerID()

	blockerMap := make(map[uuid.UUID][]uuid.UUID)
	for _, b := range blocks {
		blockerMap[b.AttackerID] = append(blockerMap[b.AttackerID], b.BlockerID)
	}

	// Check if any creature in this combat has first strike or double strike.
	// If so, we need to split into two damage sub-steps.
	hasFirstStrikeStep := false
	for _, atkID := range attackers {
		atk := g.FindPermanent(atkID)
		if atk == nil {
			continue
		}
		if atk.HasKeyword(core.FirstStrike) || atk.HasKeyword(core.DoubleStrike) {
			hasFirstStrikeStep = true
			break
		}
		for _, blkID := range blockerMap[atkID] {
			blk := g.FindPermanent(blkID)
			if blk != nil && (blk.HasKeyword(core.FirstStrike) || blk.HasKeyword(core.DoubleStrike)) {
				hasFirstStrikeStep = true
				break
			}
		}
		if hasFirstStrikeStep {
			break
		}
	}

	// Track accumulated damage on each creature across sub-steps.
	dmgTaken := make(map[uuid.UUID]int)
	// Track which creatures have been hit by deathtouch (any nonzero = lethal).
	deathtouchMarked := make(map[uuid.UUID]bool)

	var cs CombatScore

	// isDead returns true if a creature has been killed by accumulated damage
	// (including deathtouch marking from a previous step).
	isDead := func(id uuid.UUID, toughness int) bool {
		if deathtouchMarked[id] && dmgTaken[id] > 0 {
			return true
		}
		return dmgTaken[id] >= toughness
	}

	// resolveStep applies one damage sub-step. Damage within a step is simultaneous:
	// creatures that die this step still deal their damage. Only creatures killed
	// in a *previous* step are excluded.
	resolveStep := func(isFirstStrikeStep bool) {
		// Collect new damage in a separate map, then merge after the step.
		stepDmg := make(map[uuid.UUID]int)
		stepDT := make(map[uuid.UUID]bool)

		for _, atkID := range attackers {
			atk := g.FindPermanent(atkID)
			if atk == nil {
				continue
			}
			atkPow := atk.CurrentPower(g)
			atkTough := atk.CurrentToughness(g)
			atkHasFS := atk.HasKeyword(core.FirstStrike)
			atkHasDS := atk.HasKeyword(core.DoubleStrike)
			atkHasDT := atk.HasKeyword(core.Deathtouch)
			atkHasLL := atk.HasKeyword(core.Lifelink)

			blockerIDs, isBlocked := blockerMap[atkID]
			if !isBlocked || len(blockerIDs) == 0 {
				// Unblocked attacker.
				atkDealsThisStep := false
				if isFirstStrikeStep {
					atkDealsThisStep = atkHasFS || atkHasDS
				} else {
					atkDealsThisStep = !atkHasFS || atkHasDS
				}
				// Dead attackers (from a previous step) don't deal damage.
				if isDead(atkID, atkTough) {
					continue
				}
				if atkDealsThisStep && atkPow > 0 {
					cs.DamageToOpponent += atkPow
					if atkHasLL {
						cs.LifeGained += atkPow
					}
				}
				continue
			}

			// Determine if attacker deals damage this step.
			atkDealsThisStep := false
			if isFirstStrikeStep {
				atkDealsThisStep = atkHasFS || atkHasDS
			} else {
				// Normal step: creatures without first strike deal damage,
				// plus double strikers deal damage again.
				atkDealsThisStep = !atkHasFS || atkHasDS
			}

			// Dead attackers (from a previous step) don't deal damage.
			if isDead(atkID, atkTough) {
				atkDealsThisStep = false
			}

			// Attacker assigns damage to blockers.
			if atkDealsThisStep {
				remainingAtkDmg := atkPow
				atkDmgDealt := atkPow // attacker always deals its full power
				for _, blkID := range blockerIDs {
					blk := g.FindPermanent(blkID)
					if blk == nil {
						continue
					}
					blkTough := blk.CurrentToughness(g)

					// Skip blockers already dead from a previous step.
					if isDead(blkID, blkTough) {
						continue
					}

					neededToKill := blkTough - dmgTaken[blkID]
					if atkHasDT && neededToKill > 0 {
						neededToKill = 1
					}
					if neededToKill <= 0 {
						continue
					}

					assigned := neededToKill
					if assigned > remainingAtkDmg {
						assigned = remainingAtkDmg
					}
					stepDmg[blkID] += assigned
					if atkHasDT && assigned > 0 {
						stepDT[blkID] = true
					}
					remainingAtkDmg -= assigned
					if remainingAtkDmg <= 0 {
						break
					}
				}

				// Trample: excess damage goes to opponent.
				if remainingAtkDmg > 0 && atk.HasKeyword(core.Trample) {
					cs.DamageToOpponent += remainingAtkDmg
				}

				// Lifelink: attacker gains life for all damage dealt.
				if atkHasLL && atkDmgDealt > 0 {
					cs.LifeGained += atkDmgDealt
				}
			}

			// Blockers deal damage to attacker this step.
			for _, blkID := range blockerIDs {
				blk := g.FindPermanent(blkID)
				if blk == nil {
					continue
				}
				blkPow := blk.CurrentPower(g)
				blkTough := blk.CurrentToughness(g)
				blkHasFS := blk.HasKeyword(core.FirstStrike)
				blkHasDS := blk.HasKeyword(core.DoubleStrike)
				blkHasDT := blk.HasKeyword(core.Deathtouch)
				blkHasLL := blk.HasKeyword(core.Lifelink)

				// Skip blockers dead from a previous step (not this step — simultaneous).
				if isDead(blkID, blkTough) {
					continue
				}

				blkDealsThisStep := false
				if isFirstStrikeStep {
					blkDealsThisStep = blkHasFS || blkHasDS
				} else {
					blkDealsThisStep = !blkHasFS || blkHasDS
				}

				if blkDealsThisStep && blkPow > 0 {
					stepDmg[atkID] += blkPow
					if blkHasDT {
						stepDT[atkID] = true
					}
					if blkHasLL {
						cs.OpponentLifeGained += blkPow
					}
				}
			}
		}

		// Merge step damage into accumulated totals.
		for id, dmg := range stepDmg {
			dmgTaken[id] += dmg
		}
		for id := range stepDT {
			deathtouchMarked[id] = true
		}
	}

	if hasFirstStrikeStep {
		resolveStep(true)  // first strike sub-step
		resolveStep(false) // normal damage sub-step
	} else {
		resolveStep(false) // single simultaneous step
	}

	// Tally kills from accumulated damage (including deathtouch marks).
	for _, atkID := range attackers {
		atk := g.FindPermanent(atkID)
		if atk == nil {
			continue
		}
		blockerIDs := blockerMap[atkID]
		if len(blockerIDs) == 0 {
			continue
		}
		atkTough := atk.CurrentToughness(g)
		if isDead(atkID, atkTough) {
			cs.OurCreaturesLost++
		}
		for _, blkID := range blockerIDs {
			blk := g.FindPermanent(blkID)
			if blk == nil {
				continue
			}
			blkTough := blk.CurrentToughness(g)
			if isDead(blkID, blkTough) {
				cs.TheirCreaturesLost++
			}
		}
	}

	// Compute score using creature values.
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
		atkTough := atk.CurrentToughness(g)
		if isDead(atkID, atkTough) {
			ourLostValue += eval.EvalCreatureInGame(atk, g)
		}
		for _, blkID := range blockerIDs {
			blk := g.FindPermanent(blkID)
			if blk == nil {
				continue
			}
			blkTough := blk.CurrentToughness(g)
			if isDead(blkID, blkTough) {
				theirLostValue += eval.EvalCreatureInGame(blk, g)
			}
		}
	}

	cs.Score = cs.DamageToOpponent + theirLostValue - ourLostValue + cs.LifeGained - cs.OpponentLifeGained
	_ = oppID

	return cs
}

func findGangBlocks(atk *mage.Permanent, available []*mage.Permanent, g *mage.Game, playerID uuid.UUID, theyHaveLethal bool) []*mage.Permanent {
	atkTough := atk.CurrentToughness(g)
	atkScore := eval.EvalCreatureInGame(atk, g)

	// Try all 2-blocker combinations first.
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

			b1Score := eval.EvalCreatureInGame(b1, g)
			b2Score := eval.EvalCreatureInGame(b2, g)
			combinedScore := b1Score + b2Score
			// Allow gang block if attacker is worth at least 80% of blockers.
			if !theyHaveLethal && atkScore*100 < combinedScore*80 {
				continue
			}

			return []*mage.Permanent{b1, b2}
		}
	}

	// Try all 3-blocker combinations.
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

			for k := j + 1; k < len(available); k++ {
				if available[k] == nil {
					continue
				}
				b3 := available[k]
				if !mage.CanBlock(b3, atk, g) {
					continue
				}

				combinedPow := b1.CurrentPower(g) + b2.CurrentPower(g) + b3.CurrentPower(g)
				if combinedPow < atkTough {
					continue
				}

				b1Score := eval.EvalCreatureInGame(b1, g)
				b2Score := eval.EvalCreatureInGame(b2, g)
				b3Score := eval.EvalCreatureInGame(b3, g)
				combinedScore := b1Score + b2Score + b3Score
				// Allow gang block if attacker is worth at least 80% of blockers.
				if !theyHaveLethal && atkScore*100 < combinedScore*80 {
					continue
				}

				return []*mage.Permanent{b1, b2, b3}
			}
		}
	}

	return nil
}

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

func shouldAttack(atk *mage.Permanent, g *mage.Game, opponentID uuid.UUID, aggression float64) bool {
	if aggression >= 1.0 {
		return true
	}
	if profitableToAttack(atk, g, opponentID) {
		// Even if the attack is profitable against blockers, be cautious if
		// opponent has significant untapped mana (could have removal/combat tricks).
		if aggression < 0.5 {
			oppUntapped := eval.CountAvailableMana(g, opponentID)
			atkValue := eval.EvalCreatureInGame(atk, g)
			// If opponent has plenty of mana open and our creature is valuable,
			// consider holding back. High-value creatures are worth protecting.
			if oppUntapped >= 3 && atkValue >= 10 && !eval.IsEvasive(atk, opponentID, g) {
				return false
			}
		}
		return true
	}
	if aggression >= 0.7 {
		return marginallyProfitableToAttack(atk, g, opponentID)
	}
	return false
}

func marginallyProfitableToAttack(atk *mage.Permanent, g *mage.Game, opponentID uuid.UUID) bool {
	var bestBlocker *mage.Permanent
	bestPow := -1
	for _, perm := range g.Battlefield {
		if perm.Controller != opponentID || !perm.HasType(core.TypeCreature) || perm.Tapped {
			continue
		}
		if !mage.CanBlock(perm, atk, g) {
			continue
		}
		if mage.HasLandwalkEvasion(atk, opponentID, g) {
			continue
		}
		pow := perm.CurrentPower(g)
		if pow > bestPow {
			bestPow = pow
			bestBlocker = perm
		}
	}
	if bestBlocker == nil {
		return true
	}
	atkPow := atk.CurrentPower(g)
	blkTough := bestBlocker.CurrentToughness(g)
	return atkPow >= blkTough
}

func shouldBlock(atkPow int, _ *mage.Game, _ uuid.UUID, blockThreshold float64) bool {
	if blockThreshold <= 0.0 {
		return true
	}
	if blockThreshold >= 0.95 {
		return false
	}
	minPow := int(blockThreshold * 10.0)
	return atkPow >= minPow
}

// evaluateSingleBlock returns true if blocking this attacker with this blocker
// is a net-positive trade. Uses combat simulation with first strike, deathtouch,
// and lifelink awareness.
func evaluateSingleBlock(atk, blk *mage.Permanent, g *mage.Game, playerID uuid.UUID) bool {
	atkPow := atk.CurrentPower(g)
	atkTough := atk.CurrentToughness(g)
	blkPow := blk.CurrentPower(g)
	blkTough := blk.CurrentToughness(g)

	atkHasDT := atk.HasKeyword(core.Deathtouch)
	blkHasDT := blk.HasKeyword(core.Deathtouch)
	atkHasFS := atk.HasKeyword(core.FirstStrike) || atk.HasKeyword(core.DoubleStrike)
	blkHasFS := blk.HasKeyword(core.FirstStrike) || blk.HasKeyword(core.DoubleStrike)

	// Does the blocker kill the attacker?
	blockerKillsAtk := blkPow >= atkTough || (blkHasDT && blkPow > 0)
	// Does the attacker kill the blocker?
	attackerKillsBlk := atkPow >= blkTough || (atkHasDT && atkPow > 0)

	// First strike advantage: if blocker has FS and attacker doesn't,
	// and blocker kills attacker, the blocker survives.
	if blkHasFS && !atkHasFS && blockerKillsAtk {
		return true // blocker wins before attacker deals damage
	}
	// Converse: attacker has FS and blocker doesn't, attacker kills blocker
	// before blocker deals damage. Only block if we must (handled by caller).
	if atkHasFS && !blkHasFS && attackerKillsBlk && !blockerKillsAtk {
		return false // blocker dies for nothing
	}

	atkValue := eval.EvalCreatureInGame(atk, g)
	blkValue := eval.EvalCreatureInGame(blk, g)

	// If blocker kills attacker (mutual trade or blocker survives), check value trade.
	if blockerKillsAtk {
		if !attackerKillsBlk {
			return true // blocker survives, attacker dies
		}
		// Mutual trade: profitable if attacker is worth at least as much.
		return atkValue >= blkValue
	}

	// Blocker doesn't kill attacker but survives: damage soak (reduces damage to face).
	if !attackerKillsBlk {
		return atkPow >= 3 // worth chump-blocking big attackers to save life
	}

	// Blocker dies without killing attacker: only if damage prevented is significant.
	return atkPow >= 4 || atkPow*2 >= g.GetPlayer(playerID).Life()
}

func profitableToAttack(atk *mage.Permanent, g *mage.Game, opponentID uuid.UUID) bool {
	var bestBlocker *mage.Permanent
	bestPow := -1
	for _, perm := range g.Battlefield {
		if perm.Controller != opponentID || !perm.HasType(core.TypeCreature) || perm.Tapped {
			continue
		}
		if !mage.CanBlock(perm, atk, g) {
			continue
		}
		if mage.HasLandwalkEvasion(atk, opponentID, g) {
			continue
		}
		pow := perm.CurrentPower(g)
		if pow > bestPow {
			bestPow = pow
			bestBlocker = perm
		}
	}

	if bestBlocker == nil {
		return true
	}

	atkPow := atk.CurrentPower(g)
	atkTough := atk.CurrentToughness(g)
	blkPow := bestBlocker.CurrentPower(g)
	blkTough := bestBlocker.CurrentToughness(g)

	atkSurvives := blkPow < atkTough
	blkDies := atkPow >= blkTough

	if atkSurvives {
		return true
	}
	if blkDies {
		return bestBlocker.Card.ManaCost().CMC() >= atk.Card.ManaCost().CMC()
	}
	return false
}

func holdBackValue(p mage.Player, g *mage.Game, w WeightedPersonality) float64 {
	playerID := p.PlayerID()

	bestInstantValue := 0.0
	for _, card := range p.Hand() {
		if !card.HasType(core.TypeInstant) {
			continue
		}
		if !g.CanAfford(playerID, card.ManaCost()) {
			continue
		}
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
		sv := float64(eval.SpellValue(card, p, g))
		for _, a := range card.Abilities() {
			if sa, ok := a.(*mage.SpellAbility); ok {
				outcome := mage.SpellOutcome(sa.Effects())
				if outcome == mage.OutcomeDetriment {
					sv *= 1.5
				}
				if outcome == mage.OutcomeBenefit {
					sv *= 1.3
				}
			}
		}
		if sv > bestInstantValue {
			bestInstantValue = sv
		}
	}

	bestSorceryValue := 0.0
	for _, card := range g.GetCastableSpells(playerID) {
		if card.HasType(core.TypeInstant) {
			continue
		}
		sv := float64(eval.SpellValue(card, p, g))
		if sv > bestSorceryValue {
			bestSorceryValue = sv
		}
	}

	if w.HoldInstants <= 0 {
		return 0
	}

	threshold := bestSorceryValue / w.HoldInstants
	if bestInstantValue >= threshold && bestInstantValue > 0 {
		return bestInstantValue - threshold
	}
	return 0
}

func (s *HeuristicStrategy) evaluateResponse(p mage.Player, g *mage.Game) *interactive.PriorityAction {
	playerID := p.PlayerID()
	opponent := g.GetOpponent(playerID)
	if opponent == nil {
		return nil
	}

	stackHasThreat := false
	if len(g.Stack.Objects()) > 0 {
		for _, obj := range g.Stack.Objects() {
			if obj.Controller != playerID {
				stackHasThreat = true
				break
			}
		}
	}

	// Detect combat phase — instants are more valuable during combat.
	inCombat := g.Step == core.DeclareAttackers || g.Step == core.DeclareBlockers ||
		g.Step == core.CombatDamage || g.Step == core.FirstStrikeDamage

	var bestAction *interactive.PriorityAction
	bestValue := 0

	for _, card := range p.Hand() {
		if !card.HasType(core.TypeInstant) {
			continue
		}
		if !g.CanAfford(playerID, card.ManaCost()) {
			continue
		}

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

		sv := eval.SpellValue(card, p, g)

		if stackHasThreat {
			sv += 3
		}

		for _, a := range card.Abilities() {
			sa, ok := a.(*mage.SpellAbility)
			if !ok {
				continue
			}
			outcome := mage.SpellOutcome(sa.Effects())
			if outcome == mage.OutcomeDetriment {
				sv += 2
				// Removal during combat is especially valuable.
				if inCombat {
					sv += 3
				}
			}
			// Combat tricks (buff spells) are best during combat.
			if outcome == mage.OutcomeBenefit && inCombat {
				sv += 4
			}
		}

		if sv > bestValue {
			targets := s.autoSelectTargets(p, g, card)
			if len(targets) > 0 {
				bestValue = sv
				bestAction = &interactive.PriorityAction{
					Type:     interactive.ActionCastSpell,
					CardID:   card.ID(),
					CardName: card.Name(),
					Targets:  targets,
					XValue:   bestXValue(g, playerID, card, targets),
				}
			}
		}
	}

	if bestAction != nil && bestValue >= 3 {
		return bestAction
	}

	return nil
}

func raceInformedAttack(perm *mage.Permanent, g *mage.Game, opponentID uuid.UUID, race eval.RaceInfo) bool {
	if !race.Racing {
		return true
	}
	if race.MyClock < race.TheirClock {
		return true
	}
	if race.TheirClock < race.MyClock {
		return eval.IsEvasive(perm, opponentID, g)
	}
	if eval.IsEvasive(perm, opponentID, g) {
		return true
	}
	return profitableToAttack(perm, g, opponentID)
}

func raceInformedBlock(atk *mage.Permanent, blk *mage.Permanent, g *mage.Game, race eval.RaceInfo) bool {
	if !race.Racing {
		return true
	}

	atkPow := atk.CurrentPower(g)

	if race.MyClock < race.TheirClock {
		me := g.GetPlayer(blk.Controller)
		if me != nil && atkPow*4 < me.Life() {
			return false
		}
		return true
	}

	if race.TheirClock < race.MyClock {
		blkPow := blk.CurrentPower(g)
		atkTough := atk.CurrentToughness(g)
		if blkPow >= atkTough {
			return true
		}
		return true
	}

	blkPow := blk.CurrentPower(g)
	atkTough := atk.CurrentToughness(g)
	blkTough := blk.CurrentToughness(g)
	if blkPow >= atkTough {
		return true
	}
	if atkPow < blkTough {
		return true
	}
	return false
}
