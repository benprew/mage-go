package ai

import (
	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/interactive"
	"github.com/mage/mage/pkg/mage/interactive/eval"
)

// CombatScore summarizes the outcome of a combat step.
type CombatScore struct {
	DamageToOpponent   int
	OurCreaturesLost   int
	TheirCreaturesLost int
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
			if atkPow > 0 {
				cs.DamageToOpponent += atkPow
			}
			continue
		}

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

			if remainingAtkDmg >= blkTough {
				cs.TheirCreaturesLost++
				remainingAtkDmg -= blkTough
			} else {
				remainingAtkDmg = 0
			}
		}

		if remainingAtkDmg > 0 && atk.HasKeyword(core.Trample) {
			cs.DamageToOpponent += remainingAtkDmg
		}

		if totalBlockerDmg >= atkTough {
			cs.OurCreaturesLost++
		}
	}

	cs.Score = cs.DamageToOpponent +
		cs.TheirCreaturesLost*6 -
		cs.OurCreaturesLost*6

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
				theirLostValue += eval.EvalCreature(blk)
				remainingDmg -= blkTough
			}
		}
		if totalBlockerDmg >= atkTough {
			ourLostValue += eval.EvalCreature(atk)
		}
	}

	cs.Score = cs.DamageToOpponent + theirLostValue - ourLostValue
	_ = oppID

	return cs
}

func findGangBlocks(atk *mage.Permanent, available []*mage.Permanent, g *mage.Game, playerID uuid.UUID, theyHaveLethal bool) []*mage.Permanent {
	atkTough := atk.CurrentToughness(g)
	atkScore := eval.EvalCreature(atk)

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

			b1Score := eval.EvalCreature(b1)
			b2Score := eval.EvalCreature(b2)
			if !theyHaveLethal && atkScore < b1Score+b2Score {
				continue
			}

			return []*mage.Permanent{b1, b2}
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
