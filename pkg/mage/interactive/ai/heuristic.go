package ai

import (
	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/ai/combatsolver"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/eval"
)

// stackHasOpponentThreat returns true when a stack object controlled by
// someone other than playerID is currently resolving — typically an
// opponent's spell or ability that may threaten our creatures.
func stackHasOpponentThreat(g *mage.Game, playerID uuid.UUID) bool {
	for _, obj := range g.StackObjects() {
		if obj.Controller != playerID {
			return true
		}
	}
	return false
}

// shouldHoldForCombat returns true when the AI should defer casting combat-
// eligible instants (pump, damage, removal) to the post-blockers response
// window: it's the AI's pre-combat main phase and they have at least one
// creature that can attack.
func (s *HeuristicStrategy) shouldHoldForCombat(g *mage.Game, playerID uuid.UUID) bool {
	if g.GetStep() != core.PrecombatMain {
		return false
	}
	if g.ActivePlayerObj() == nil || g.ActivePlayerObj().PlayerID() != playerID {
		return false
	}
	for _, perm := range g.AllBattlefield() {
		if perm.Controller == playerID && perm.CanDeclareAsAttacker(g) {
			return true
		}
	}
	return false
}

// solverProfile translates a WeightedPersonality into a combatsolver.Profile.
func (s *HeuristicStrategy) solverProfile() combatsolver.Profile {
	w := s.weights()
	return combatsolver.Profile{
		Weights:        w.Weights,
		Aggression:     w.Aggression,
		BlockThreshold: w.BlockThreshold,
	}
}

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

	// Stack-threat exception: if an opponent's spell or ability is on the
	// stack (e.g., a kill spell targeting one of our creatures), evaluate a
	// response immediately — even during our main phase — so a held protection
	// or pump trick can save the targeted creature instead of being suppressed
	// by the combat-trick hold logic below.
	if !mainPhase || stackHasOpponentThreat(g, playerID) {
		if response := s.evaluateResponse(p, g); response != nil {
			return *response
		}
	}

	if mainPhase {
		holdCombatTricks := s.shouldHoldForCombat(g, playerID)
		for _, card := range p.Hand() {
			if !card.HasType(core.TypeInstant) {
				continue
			}
			if !g.CanAfford(playerID, card.ManaCost()) {
				continue
			}
			// Hold combat-eligible instants for the post-blockers response
			// window when we're pre-combat and have attackable creatures.
			if holdCombatTricks && combatsolver.ClassifyCombat(card) != combatsolver.RoleNone {
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

		// Pick the friendliness of the desired target from the ability's
		// dominant outcome: detrimental abilities should hit opponents,
		// beneficial ones should hit our own permanents.
		preferOwn := mage.SpellOutcome(ab.Effects()) == mage.OutcomeBenefit

		var targets []uuid.UUID
		for _, t := range ab.Targets() {
			possible := t.Possible(playerID, perm.Card, g)
			if len(possible) == 0 {
				return nil
			}
			opponent := g.GetOpponent(playerID)
			bestTarget := possible[0]
			if preferOwn {
				for _, id := range possible {
					p := g.FindPermanent(id)
					if p != nil && p.Controller == playerID {
						bestTarget = id
						break
					}
				}
			} else if opponent != nil {
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

	// Lethal short-circuit: if a known lethal attack exists, take it.
	lethal := eval.CalculateLethal(g, playerID)
	if lethal.IHaveLethal && len(lethal.LethalAttackers) > 0 {
		return lethal.LethalAttackers
	}

	// Joint-optimal solver: enumerate attacker subsets, pick the one with the
	// best outcome under opponent's optimal blocking response. Returns nil
	// (skip combat) when no positive line exists.
	r := combatsolver.SolveAttack(g, playerID, combatsolver.Options{Profile: s.solverProfile()})
	return r.Attackers
}

func (s *HeuristicStrategy) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	r := combatsolver.SolveDefense(g, p.PlayerID(), combatsolver.Options{Profile: s.solverProfile()})
	return r.Blocks
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
