package ai

import (
	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/eval"
	"github.com/google/uuid"
)

// HeuristicStrategy implements AIStrategy using personality-driven heuristics.
type HeuristicStrategy struct {
	Personality Personality
	Weights     WeightedPersonality
	weightsInit bool
}

// NewHeuristicStrategy creates a HeuristicStrategy from a WeightedPersonality.
func NewHeuristicStrategy(w WeightedPersonality) *HeuristicStrategy {
	return &HeuristicStrategy{Weights: w, weightsInit: true}
}

func newHeuristicFromOld(p Personality) *HeuristicStrategy {
	return &HeuristicStrategy{Personality: p, Weights: p.ToWeighted(), weightsInit: true}
}

func (s *HeuristicStrategy) weights() WeightedPersonality {
	if !s.weightsInit {
		s.Weights = s.Personality.ToWeighted()
		s.weightsInit = true
	}
	return s.Weights
}

func (s *HeuristicStrategy) PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) interactive.PriorityAction {
	playerID := p.PlayerID()

	lethal := eval.CalculateLethal(g, playerID)
	if lethal.TheyHaveLethal {
		if removal := s.findBestRemoval(p, g); removal != nil {
			return *removal
		}
	}

	if mainPhase {
		if lands := g.GetPlayableLands(playerID); len(lands) > 0 {
			if bestLand := chooseBestLand(p, g); bestLand != nil {
				return interactive.PriorityAction{
					Type:     interactive.ActionPlayLand,
					CardID:   bestLand.ID(),
					CardName: bestLand.Name(),
				}
			}
		}

		if holdBackValue(p, g, s.weights()) > 0 {
			return interactive.PriorityAction{Type: interactive.ActionPass}
		}

		availMana := eval.CountAvailableMana(g, playerID)
		cmcs := eval.HandCMCs(p.Hand())

		var bestCard mage.Card
		bestScore := -1
		for _, card := range g.GetCastableSpells(playerID) {
			if card.HasType(core.TypeInstant) {
				continue
			}
			score := eval.SpellValue(card, p, g)
			curveBonus := eval.ManaCurveBonus(card.ManaCost().CMC(), availMana, cmcs)
			score += int(curveBonus)
			if score > bestScore {
				bestScore = score
				bestCard = card
			}
		}
		if bestCard != nil && !eval.SpellIsWorthless(bestCard, p, g) {
			targets := s.autoSelectTargets(p, g, bestCard)
			return interactive.PriorityAction{
				Type:     interactive.ActionCastSpell,
				CardID:   bestCard.ID(),
				CardName: bestCard.Name(),
				Targets:  targets,
				XValue:   bestXValue(g, playerID, bestCard, targets),
			}
		}
	}

	if action := s.considerAbilityActivation(p, g); action != nil {
		return *action
	}

	if !mainPhase {
		if response := s.evaluateResponse(p, g); response != nil {
			return *response
		}
	}

	if mainPhase {
		for _, card := range p.Hand() {
			if !card.HasType(core.TypeInstant) {
				continue
			}
			if !g.CanAfford(playerID, card.ManaCost()) {
				continue
			}
			hasUsableEffect := false
			for _, a := range card.Abilities() {
				if sa, ok := a.(*mage.SpellAbility); ok && sa.Kind() == mage.ActionSpell {
					if mage.SpellOutcome(sa.Effects()) != mage.OutcomeUnknown {
						hasUsableEffect = true
						break
					}
				}
			}
			if hasUsableEffect {
				targets := s.autoSelectTargets(p, g, card)
				if len(targets) > 0 {
					return interactive.PriorityAction{
						Type:     interactive.ActionCastSpell,
						CardID:   card.ID(),
						CardName: card.Name(),
						Targets:  targets,
						XValue:   bestXValue(g, playerID, card, targets),
					}
				}
			}
		}
	}

	return interactive.PriorityAction{Type: interactive.ActionPass}
}

func (s *HeuristicStrategy) findBestRemoval(p mage.Player, g *mage.Game) *interactive.PriorityAction {
	playerID := p.PlayerID()
	opponent := g.GetOpponent(playerID)
	if opponent == nil {
		return nil
	}

	for _, card := range p.Hand() {
		if !g.CanAfford(playerID, card.ManaCost()) {
			continue
		}
		for _, a := range card.Abilities() {
			sa, ok := a.(*mage.SpellAbility)
			if !ok || sa.Kind() != mage.ActionSpell {
				continue
			}
			outcome := mage.SpellOutcome(sa.Effects())
			if outcome == mage.OutcomeDetriment {
				targets := s.autoSelectTargets(p, g, card)
				if len(targets) > 0 {
					for _, tid := range targets {
						perm := g.FindPermanent(tid)
						if perm != nil && perm.Controller == opponent.PlayerID() {
							return &interactive.PriorityAction{
								Type:     interactive.ActionCastSpell,
								CardID:   card.ID(),
								CardName: card.Name(),
								Targets:  targets,
								XValue:   bestXValue(g, playerID, card, targets),
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func (s *HeuristicStrategy) considerAbilityActivation(p mage.Player, g *mage.Game) *interactive.PriorityAction {
	playerID := p.PlayerID()
	abilities := g.GetActivatableAbilities(playerID)
	if len(abilities) == 0 {
		return nil
	}

	var bestInfo *mage.ActivatableInfo
	bestScore := 0

	for i := range abilities {
		info := &abilities[i]
		perm := g.FindPermanent(info.PermanentID)
		if perm == nil {
			continue
		}
		if info.AbilityIndex < 0 || info.AbilityIndex >= len(perm.RuntimeAbilities) {
			continue
		}
		ab, ok := mage.UnwrapAbility(perm.RuntimeAbilities[info.AbilityIndex]).(mage.ActivatedAbility)
		if !ok {
			continue
		}
		score := eval.AbilityQuality(ab)
		if score > bestScore {
			bestScore = score
			bestInfo = info
		}
	}

	if bestInfo != nil && bestScore >= 3 {
		perm := g.FindPermanent(bestInfo.PermanentID)
		if perm == nil {
			return nil
		}
		ab, ok := mage.UnwrapAbility(perm.RuntimeAbilities[bestInfo.AbilityIndex]).(mage.ActivatedAbility)
		if !ok {
			return nil
		}

		var targets []uuid.UUID
		for _, t := range ab.Targets() {
			possible := t.Possible(playerID, perm.Card, g)
			if len(possible) == 0 {
				return nil
			}
			opponent := g.GetOpponent(playerID)
			bestTarget := possible[0]
			if opponent != nil {
				for _, id := range possible {
					p := g.FindPermanent(id)
					if p != nil && p.Controller == opponent.PlayerID() {
						bestTarget = id
						break
					}
				}
			}
			targets = append(targets, bestTarget)
		}

		return &interactive.PriorityAction{
			Type:         interactive.ActionActivateAbility,
			PermanentID:  bestInfo.PermanentID,
			AbilityIndex: bestInfo.AbilityIndex,
			Targets:      targets,
		}
	}

	return nil
}

func (s *HeuristicStrategy) Attackers(p mage.Player, g *mage.Game) []uuid.UUID {
	playerID := p.PlayerID()
	opponent := g.GetOpponent(playerID)
	var opponentID uuid.UUID
	if opponent != nil {
		opponentID = opponent.PlayerID()
	}

	lethal := eval.CalculateLethal(g, playerID)
	if lethal.IHaveLethal && len(lethal.LethalAttackers) > 0 {
		return lethal.LethalAttackers
	}

	race := eval.CalculateRace(g, playerID)

	var attackers []uuid.UUID
	for _, perm := range g.AllBattlefield() {
		if perm.Controller != playerID {
			continue
		}
		if !perm.CanDeclareAsAttacker(g) {
			continue
		}

		if race.Racing {
			if raceInformedAttack(perm, g, opponentID, race) {
				attackers = append(attackers, perm.ID())
			}
		} else if shouldAttack(perm, g, opponentID, s.weights().Aggression) {
			attackers = append(attackers, perm.ID())
		}
	}
	return attackers
}

func (s *HeuristicStrategy) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	playerID := p.PlayerID()

	lethal := eval.CalculateLethal(g, playerID)
	theyHaveLethal := lethal.TheyHaveLethal

	race := eval.CalculateRace(g, playerID)

	var assignments []mage.BlockAssignment

	var available []*mage.Permanent
	for _, perm := range g.AllBattlefield() {
		if perm.Controller != playerID || !perm.CanDeclareAsBlocker(g) {
			continue
		}
		available = append(available, perm)
	}

	singleBlockedAttackers := make(map[uuid.UUID]bool)

	for _, group := range g.CombatGroups() {
		if group.DefenderID != playerID {
			continue
		}
		atk := g.FindPermanent(group.AttackerID)
		if atk == nil {
			continue
		}
		atkPow := atk.CurrentPower(g)

		if !theyHaveLethal {
			if race.Racing && race.MyClock < race.TheirClock {
				me := g.GetPlayer(playerID)
				if me != nil && atkPow*4 < me.Life() {
					continue
				}
			}
			if !shouldBlock(atkPow, g, p.PlayerID(), s.weights().BlockThreshold) {
				continue
			}
		}

		assigned := false
		for i, blk := range available {
			if blk == nil {
				continue
			}
			if !mage.CanBlock(blk, atk, g) {
				continue
			}
			if mage.HasLandwalkEvasion(atk, playerID, g) {
				continue
			}

			if theyHaveLethal {
				assignments = append(assignments, mage.BlockAssignment{
					BlockerID:  blk.ID(),
					AttackerID: atk.ID(),
				})
				available[i] = nil
				assigned = true
				break
			}

			if race.Racing && !raceInformedBlock(atk, blk, g, race) {
				continue
			}

			if evaluateSingleBlock(atk, blk, g, playerID) {
				assignments = append(assignments, mage.BlockAssignment{
					BlockerID:  blk.ID(),
					AttackerID: atk.ID(),
				})
				available[i] = nil
				assigned = true
				singleBlockedAttackers[atk.ID()] = true
				break
			}
		}
		_ = assigned
	}

	for _, group := range g.CombatGroups() {
		if group.DefenderID != playerID {
			continue
		}
		atk := g.FindPermanent(group.AttackerID)
		if atk == nil {
			continue
		}
		alreadyBlocked := false
		for _, a := range assignments {
			if a.AttackerID == atk.ID() {
				alreadyBlocked = true
				break
			}
		}
		if alreadyBlocked {
			continue
		}
		if canSingleBlockKill(atk, available, g) {
			continue
		}
		if mage.HasLandwalkEvasion(atk, playerID, g) {
			continue
		}

		gangBlockers := findGangBlocks(atk, available, g, playerID, theyHaveLethal)
		if gangBlockers == nil {
			continue
		}

		for _, blk := range gangBlockers {
			assignments = append(assignments, mage.BlockAssignment{
				BlockerID:  blk.ID(),
				AttackerID: atk.ID(),
			})
			for i, a := range available {
				if a != nil && a.ID() == blk.ID() {
					available[i] = nil
					break
				}
			}
		}
	}

	return assignments
}

func (s *HeuristicStrategy) autoSelectTargets(p mage.Player, g *mage.Game, card mage.Card) []uuid.UUID {
	playerID := p.PlayerID()

	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok || sa.Kind() != mage.ActionSpell {
			continue
		}
		for _, t := range sa.Targets() {
			possible := t.Possible(playerID, card, g)
			if len(possible) == 0 {
				return nil
			}

			outcome := mage.SpellOutcome(sa.Effects())
			switch t.(type) {
			case *mage.AnyTarget:
				if outcome == mage.OutcomeBenefit {
					var ownBest uuid.UUID
					ownBestScore := -1
					for _, id := range possible {
						perm := g.FindPermanent(id)
						if perm != nil && perm.Controller == playerID && perm.HasType(core.TypeCreature) {
							score := eval.EvalCreatureInGame(perm, g)
							if score > ownBestScore {
								ownBestScore = score
								ownBest = id
							}
						}
					}
					if ownBest != uuid.Nil {
						return []uuid.UUID{ownBest}
					}
					for _, id := range possible {
						if id == playerID {
							return []uuid.UUID{id}
						}
					}
				}
				opponent := g.GetOpponent(playerID)
				if s.weights().TargetFace >= 0.5 && opponent != nil {
					for _, id := range possible {
						if id == opponent.PlayerID() {
							return []uuid.UUID{id}
						}
					}
				}
				spellDamage := 0
				for _, ab := range card.Abilities() {
					if sa, ok := ab.(*mage.SpellAbility); ok && sa.Kind() == mage.ActionSpell {
						for _, e := range sa.Effects() {
							if dv := e.Properties().DamageValue; dv != nil {
								spellDamage = dv.Resolve(g, card.ID(), playerID, nil)
							}
						}
					}
				}
				if spellDamage > 0 && opponent != nil {
					var bestLethalID uuid.UUID
					bestLethalTPM := -1.0
					for _, id := range possible {
						perm := g.FindPermanent(id)
						if perm != nil && perm.Controller == opponent.PlayerID() && perm.HasType(core.TypeCreature) {
							if spellDamage >= perm.CurrentToughness(g) {
								tpm := eval.ThreatPerManaInGame(perm, g)
								if tpm > bestLethalTPM {
									bestLethalTPM = tpm
									bestLethalID = id
								}
							}
						}
					}
					if bestLethalID != uuid.Nil {
						return []uuid.UUID{bestLethalID}
					}
				}
				var bestID uuid.UUID
				bestScore := -1
				for _, id := range possible {
					perm := g.FindPermanent(id)
					if perm != nil && perm.Controller != playerID && perm.HasType(core.TypeCreature) {
						score := eval.EvalCreatureInGame(perm, g)
						if score > bestScore {
							bestScore = score
							bestID = id
						}
					}
				}
				if bestID != uuid.Nil {
					return []uuid.UUID{bestID}
				}
				if opponent != nil {
					return []uuid.UUID{opponent.PlayerID()}
				}

			case *mage.CreatureTarget:
				if outcome == mage.OutcomeBenefit {
					var ownBest uuid.UUID
					ownBestScore := -1
					for _, id := range possible {
						perm := g.FindPermanent(id)
						if perm != nil && perm.Controller == playerID {
							score := eval.EvalCreatureInGame(perm, g)
							if score > ownBestScore {
								ownBestScore = score
								ownBest = id
							}
						}
					}
					if ownBest != uuid.Nil {
						return []uuid.UUID{ownBest}
					}
				}
				var bestID uuid.UUID
				bestScore := -1
				for _, id := range possible {
					perm := g.FindPermanent(id)
					if perm != nil && perm.Controller != playerID {
						score := eval.EvalCreatureInGame(perm, g)
						if score > bestScore {
							bestScore = score
							bestID = id
						}
					}
				}
				if bestID != uuid.Nil {
					return []uuid.UUID{bestID}
				}
				// Only fall back to own creatures for beneficial spells;
				// never target your own creature with removal.
				if outcome == mage.OutcomeBenefit {
					for _, id := range possible {
						perm := g.FindPermanent(id)
						if perm != nil && perm.Controller == playerID {
							return []uuid.UUID{id}
						}
					}
				}

			case *mage.PlayerTarget, *mage.OpponentTarget:
				opponent := g.GetOpponent(playerID)
				if opponent != nil {
					return []uuid.UUID{opponent.PlayerID()}
				}

			default:
				if len(possible) > 0 {
					return []uuid.UUID{possible[0]}
				}
			}
		}
	}

	// Check CastTargets for auras and other cards with cast-time targeting
	for _, t := range card.CastTargets() {
		possible := t.Possible(playerID, card, g)
		if len(possible) == 0 {
			return nil
		}
		switch t.(type) {
		case *mage.CreatureTarget:
			var ownBest uuid.UUID
			ownBestScore := -1
			for _, id := range possible {
				perm := g.FindPermanent(id)
				if perm != nil && perm.Controller == playerID {
					score := eval.EvalCreatureInGame(perm, g)
					if score > ownBestScore {
						ownBestScore = score
						ownBest = id
					}
				}
			}
			if ownBest != uuid.Nil {
				return []uuid.UUID{ownBest}
			}
			for _, id := range possible {
				perm := g.FindPermanent(id)
				if perm != nil {
					return []uuid.UUID{id}
				}
			}
		default:
			if len(possible) > 0 {
				return []uuid.UUID{possible[0]}
			}
		}
	}

	return nil
}

// chooseBestLand selects the best land to play from hand.
// Prefers lands that provide colors needed by spells in hand.
// bestXValue picks the best X value for an X-cost spell.
// For damage spells it tries lethal values; otherwise it spends all available mana.
func bestXValue(g *mage.Game, playerID uuid.UUID, card mage.Card, targets []uuid.UUID) int {
	mc := card.ManaCost()
	if !mc.HasX {
		return 0
	}
	fixedCost := mc.CMC()
	availMana := eval.CountAvailableMana(g, playerID)
	maxX := availMana - fixedCost
	if maxX < 1 {
		return 1
	}

	// For damage spells, try to pick a lethal X value.
	isDamageSpell := false
	for _, a := range card.Abilities() {
		if sa, ok := a.(*mage.SpellAbility); ok && sa.Kind() == mage.ActionSpell {
			for _, e := range sa.Effects() {
				if e.Properties().DamageValue != nil {
					isDamageSpell = true
					break
				}
			}
		}
	}

	if isDamageSpell && len(targets) > 0 {
		// If targeting a player, try lethal.
		if tp := g.GetPlayer(targets[0]); tp != nil && tp.PlayerID() != playerID {
			if life := tp.Life(); life > 0 && life <= maxX {
				return life
			}
		}
		// If targeting a creature, try exact toughness.
		if perm := g.FindPermanent(targets[0]); perm != nil && perm.Controller != playerID {
			if tough := perm.CurrentToughness(g); tough > 0 && tough <= maxX {
				return tough
			}
		}
	}

	// Default: spend all available mana.
	return maxX
}

func chooseBestLand(p mage.Player, _ *mage.Game) mage.Card {
	hand := p.Hand()

	var lands []mage.Card
	neededColors := make(map[core.Color]int)

	for _, c := range hand {
		if c.HasType(core.TypeLand) {
			lands = append(lands, c)
		} else {
			mc := c.ManaCost()
			neededColors[core.White] += mc.White
			neededColors[core.Blue] += mc.Blue
			neededColors[core.Black] += mc.Black
			neededColors[core.Red] += mc.Red
			neededColors[core.Green] += mc.Green
		}
	}

	if len(lands) == 0 {
		return nil
	}
	if len(lands) == 1 {
		return lands[0]
	}

	// Score each land by how many needed colors it produces.
	var bestLand mage.Card
	bestScore := -1

	for _, land := range lands {
		score := 0
		for _, a := range land.Abilities() {
			ma, ok := mage.UnwrapAbility(a).(*mage.ManaAbility)
			if !ok {
				continue
			}
			if ma.HasAnyColor() {
				for _, need := range neededColors {
					score += need
				}
			} else if ma.PrimaryColor() != core.Colorless {
				score += neededColors[ma.PrimaryColor()] * 2
			}
		}
		if score > bestScore {
			bestScore = score
			bestLand = land
		}
	}

	if bestLand != nil {
		return bestLand
	}
	return lands[0]
}
