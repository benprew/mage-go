package heuristic

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai/combatsolver"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

// baselineStrategy is an exact copy of the heuristic AI strategy before our improvements.
type baselineStrategy struct {
	Weights ai.WeightedPersonality
}

func newBaselineStrategy(wp ai.WeightedPersonality) ai.AIStrategy {
	return &baselineStrategy{Weights: wp}
}

func (s *baselineStrategy) solverProfile() combatsolver.Profile {
	weights := s.Weights.Weights
	weights.Life += 1.0
	weights.Board *= 0.85
	return combatsolver.Profile{
		Weights:        weights,
		Aggression:     s.Weights.Aggression,
		BlockThreshold: s.Weights.BlockThreshold,
	}
}

func (s *baselineStrategy) Attackers(p mage.Player, g *mage.Game) []uuid.UUID {
	playerID := p.PlayerID()
	lethal := eval.CalculateLethal(g, playerID)
	if lethal.IHaveLethal && len(lethal.LethalAttackers) > 0 {
		return lethal.LethalAttackers
	}
	r := combatsolver.SolveAttack(g, playerID, combatsolver.Options{Profile: s.solverProfile()})
	return r.Attackers
}

func (s *baselineStrategy) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	r := combatsolver.SolveDefense(g, p.PlayerID(), combatsolver.Options{Profile: s.solverProfile()})
	return r.Blocks
}

func (s *baselineStrategy) PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) interactive.PriorityAction {
	playerID := p.PlayerID()

	lethal := eval.CalculateLethal(g, playerID)
	if lethal.TheyHaveLethal {
		if removal := s.findBestRemoval(p, g); removal != nil {
			return *removal
		}
	}

	if g.GetStep() == core.DeclareBlockers {
		if action := s.considerRegeneration(p, g); action != nil {
			return *action
		}
		if action := s.considerCombatPump(p, g); action != nil {
			return *action
		}
	}

	if stackHasOpponentThreat(g, playerID) {
		if action := s.considerRegenerationAgainstRemoval(p, g); action != nil {
			return *action
		}
		if action := s.considerPumpAgainstRemoval(p, g); action != nil {
			return *action
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

		heldInstant := instantToHold(p, g, s.Weights)
		availMana := eval.CountAvailableMana(g, playerID)
		cmcs := eval.HandCMCs(p.Hand())

		if action := s.bestSpellAction(p, g, g.GetCastableSpells(playerID), func(card mage.Card) (int, bool) {
			if card.HasType(core.TypeInstant) {
				return 0, false
			}
			if heldInstant != nil && (card.ManaCost().HasX || !canCastWhileReserving(g, playerID, card, heldInstant)) {
				return 0, false
			}
			score := eval.SpellValue(card, p, g)
			score += oldManaCurveBonus(card.ManaCost().CMC(), availMana, cmcs)
			return score, true
		}); action != nil {
			return *action
		}

		if heldInstant != nil {
			return interactive.PriorityAction{Type: interactive.ActionPass}
		}
	}

	if action := s.considerAbilityActivation(p, g); action != nil {
		return *action
	}

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
			if !g.CanAfford(playerID, card.ManaCost(), mage.SpellContextForCard(card)) {
				continue
			}
			if holdCombatTricks && combatsolver.ClassifyCombat(card) != combatsolver.RoleNone {
				continue
			}
			if !aiHintAllowsTiming(cardAIHint(card), g, mainPhase) {
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
				if !requiresTargets(card) || len(targets) > 0 {
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

	_ = landsPlayed
	return interactive.PriorityAction{Type: interactive.ActionPass}
}

func (s *baselineStrategy) shouldHoldForCombat(g *mage.Game, playerID uuid.UUID) bool {
	if g.GetStep() != core.PrecombatMain {
		return false
	}
	if g.ActivePlayerObj() == nil || g.ActivePlayerObj().PlayerID() != playerID {
		return false
	}
	for _, perm := range g.AllBattlefield() {
		if perm.ControllerID() == playerID && perm.CanDeclareAsAttacker(g) {
			return true
		}
	}
	return false
}

func (s *baselineStrategy) bestSpellAction(p mage.Player, g *mage.Game, cards []mage.Card, scoreCard func(mage.Card) (int, bool)) *interactive.PriorityAction {
	playerID := p.PlayerID()
	var bestAction *interactive.PriorityAction
	bestScore := 0

	for _, card := range cards {
		score, ok := scoreCard(card)
		if !ok || score <= 0 || eval.SpellIsWorthless(card, p, g) {
			continue
		}
		if !aiHintAllowsTiming(cardAIHint(card), g, g.GetStep().IsMainPhase()) {
			continue
		}
		targets := s.autoSelectTargets(p, g, card)
		if requiresTargets(card) && len(targets) == 0 {
			continue
		}
		if score <= bestScore {
			continue
		}
		bestScore = score
		bestAction = &interactive.PriorityAction{
			Type:     interactive.ActionCastSpell,
			CardID:   card.ID(),
			CardName: card.Name(),
			Targets:  targets,
			XValue:   bestXValue(g, playerID, card, targets),
		}
	}

	return bestAction
}

func (s *baselineStrategy) findBestRemoval(p mage.Player, g *mage.Game) *interactive.PriorityAction {
	st := New(s.Weights)
	return st.findBestRemoval(p, g)
}

func (s *baselineStrategy) considerRegeneration(p mage.Player, g *mage.Game) *interactive.PriorityAction {
	st := New(s.Weights)
	return st.considerRegeneration(p, g)
}

func (s *baselineStrategy) considerCombatPump(p mage.Player, g *mage.Game) *interactive.PriorityAction {
	st := New(s.Weights)
	return st.considerCombatPump(p, g)
}

func (s *baselineStrategy) considerRegenerationAgainstRemoval(p mage.Player, g *mage.Game) *interactive.PriorityAction {
	st := New(s.Weights)
	return st.considerRegenerationAgainstRemoval(p, g)
}

func (s *baselineStrategy) considerPumpAgainstRemoval(p mage.Player, g *mage.Game) *interactive.PriorityAction {
	st := New(s.Weights)
	return st.considerPumpAgainstRemoval(p, g)
}

func (s *baselineStrategy) considerAbilityActivation(p mage.Player, g *mage.Game) *interactive.PriorityAction {
	st := New(s.Weights)
	return st.considerAbilityActivation(p, g)
}

func (s *baselineStrategy) evaluateResponse(p mage.Player, g *mage.Game) *interactive.PriorityAction {
	st := New(s.Weights)
	return st.evaluateResponse(p, g)
}

func (s *baselineStrategy) autoSelectTargets(p mage.Player, g *mage.Game, card mage.Card) []uuid.UUID {
	return oldAutoSelectTargets(p, g, card, s.Weights)
}

func oldAutoSelectTargets(p mage.Player, g *mage.Game, card mage.Card, w ai.WeightedPersonality) []uuid.UUID {
	playerID := p.PlayerID()
	opponent := g.GetOpponent(playerID)
	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok || sa.Kind() != mage.ActionSpell {
			continue
		}
		for _, t := range sa.Targets() {
			possible := t.Possible(playerID, card, g)
			if len(possible) == 0 {
				continue
			}
			switch t.(type) {
			case *mage.DamageAnyTarget:
				// Old target face check (needed TargetFace >= 0.5)
				if w.TargetFace >= 0.5 && opponent != nil {
					for _, id := range possible {
						if id == opponent.PlayerID() {
							return []uuid.UUID{id}
						}
					}
				}
				for _, id := range possible {
					if perm := g.FindPermanent(id); perm != nil && perm.ControllerID() != playerID {
						return []uuid.UUID{id}
					}
				}
				if opponent != nil {
					return []uuid.UUID{opponent.PlayerID()}
				}
			case *mage.CreatureTarget:
				for _, id := range possible {
					if perm := g.FindPermanent(id); perm != nil {
						return []uuid.UUID{id}
					}
				}
			}
		}
	}
	st := New(w)
	return st.autoSelectTargets(p, g, card)
}

func oldManaCurveBonus(cardCMC, availableMana int, handCMCs []int) int {
	if availableMana <= 0 {
		return 0
	}
	maxCastable := 0
	for _, cmc := range handCMCs {
		if cmc <= availableMana && cmc > maxCastable {
			maxCastable = cmc
		}
	}
	if cardCMC == maxCastable {
		return 3
	}
	if maxCastable > 0 && cardCMC < maxCastable {
		wastedMana := availableMana - cardCMC
		if wastedMana >= 2 {
			return -2
		}
	}
	return 0
}
