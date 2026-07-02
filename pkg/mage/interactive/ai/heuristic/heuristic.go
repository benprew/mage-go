// Package heuristic implements a fast, personality-driven AI strategy that
// makes decisions via local heuristics (without search).
package heuristic

import (
	"maps"
	"slices"
	"sort"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai/combatsolver"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
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
func (s *Strategy) shouldHoldForCombat(g *mage.Game, playerID uuid.UUID) bool {
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
func (s *Strategy) solverProfile() combatsolver.Profile {
	w := s.weights()
	weights := w.Weights
	weights.Life += 1.0
	weights.Board *= 0.85
	return combatsolver.Profile{
		Weights:        weights,
		Aggression:     w.Aggression,
		BlockThreshold: w.BlockThreshold,
	}
}

// Strategy implements ai.AIStrategy using personality-driven heuristics.
type Strategy struct {
	Personality ai.Personality
	Weights     ai.WeightedPersonality
	weightsInit bool
}

// New creates a Strategy from a WeightedPersonality.
func New(w ai.WeightedPersonality) *Strategy {
	return &Strategy{Weights: w, weightsInit: true}
}

// NewFromOld creates a Strategy from a legacy boolean Personality.
func NewFromOld(p ai.Personality) *Strategy {
	return &Strategy{Personality: p, Weights: p.ToWeighted(), weightsInit: true}
}

func (s *Strategy) weights() ai.WeightedPersonality {
	if !s.weightsInit {
		s.Weights = s.Personality.ToWeighted()
		s.weightsInit = true
	}
	return s.Weights
}

func (s *Strategy) PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) interactive.PriorityAction {
	playerID := p.PlayerID()

	lethal := eval.CalculateLethal(g, playerID)
	if lethal.TheyHaveLethal {
		if removal := s.findBestRemoval(p, g); removal != nil {
			return *removal
		}
	}

	// Once blocks are declared, save a creature combat would otherwise kill by
	// installing a regeneration shield before the damage step.
	if g.GetStep() == core.DeclareBlockers {
		if action := s.considerRegeneration(p, g); action != nil {
			return *action
		}
		if action := s.considerPumpForCombatKill(p, g); action != nil {
			return *action
		}
	}

	// Save a creature from a destroy- or lethal-burn removal spell or ability on
	// the stack by installing a regeneration shield (or casting a regeneration
	// spell) in response, before the removal resolves.
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

		heldInstant := instantToHold(p, g, s.weights())

		availMana := eval.CountAvailableMana(g, playerID)
		cmcs := eval.HandCMCs(p.Hand())

		if action := s.bestSpellAction(p, g, g.GetCastableSpells(playerID), func(card mage.Card) (int, bool) {
			if card.HasType(core.TypeInstant) {
				return 0, false
			}
			// When holding an instant up (combat trick, removal, counter), keep
			// developing the board — but only with a spell we can pay for while
			// still affording the held instant afterward. This reserves mana
			// rather than passing the entire main phase. X-cost spells consume
			// all available mana, so they can never coexist with a held instant.
			if heldInstant != nil && (card.ManaCost().HasX || !canCastWhileReserving(g, playerID, card, heldInstant)) {
				return 0, false
			}
			score := eval.SpellValue(card, p, g)
			score += int(eval.ManaCurveBonus(card.ManaCost().CMC(), availMana, cmcs))
			return score, true
		}); action != nil {
			return *action
		}

		// We are holding an instant and have no board play that leaves its mana
		// up: keep the mana available instead of casting the instant proactively
		// at sorcery speed via the main-phase instant loop below.
		if heldInstant != nil {
			return interactive.PriorityAction{Type: interactive.ActionPass}
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

func (s *Strategy) bestSpellAction(p mage.Player, g *mage.Game, cards []mage.Card, scoreCard func(mage.Card) (int, bool)) *interactive.PriorityAction {
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

func (s *Strategy) findBestRemoval(p mage.Player, g *mage.Game) *interactive.PriorityAction {
	playerID := p.PlayerID()
	opponent := g.GetOpponent(playerID)
	if opponent == nil {
		return nil
	}

	for _, card := range p.Hand() {
		if !g.CanAfford(playerID, card.ManaCost(), mage.SpellContextForCard(card)) {
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

func (s *Strategy) considerAbilityActivation(p mage.Player, g *mage.Game) *interactive.PriorityAction {
	playerID := p.PlayerID()
	abilities := g.GetActivatableAbilities(playerID)
	if len(abilities) == 0 {
		return nil
	}

	var bestInfo *mage.ActivatableInfo
	var bestTargets []uuid.UUID
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
		targets, ok := s.autoSelectAbilityTargets(p, g, perm, ab)
		if !ok || ai.AbilityActivationIsRedundant(g, playerID, perm.ID(), ab.Effects(), targets) {
			continue
		}
		score := eval.AbilityQuality(ab)
		if score > bestScore {
			bestScore = score
			bestInfo = info
			bestTargets = targets
		}
	}

	if bestInfo != nil && bestScore >= 3 {
		return &interactive.PriorityAction{
			Type:         interactive.ActionActivateAbility,
			PermanentID:  bestInfo.PermanentID,
			AbilityIndex: bestInfo.AbilityIndex,
			Targets:      bestTargets,
		}
	}

	return nil
}

func (s *Strategy) autoSelectAbilityTargets(p mage.Player, g *mage.Game, perm *mage.Permanent, ab mage.ActivatedAbility) ([]uuid.UUID, bool) {
	playerID := p.PlayerID()
	purpose := eval.TargetPurposeForEffects(ab.Effects())
	hint := abilityAIHint(ab)
	if hintedPurpose := eval.TargetPurposeFromAI(hint.TargetPurpose); hintedPurpose != eval.TargetGeneric {
		purpose = hintedPurpose
	}
	damage := abilityDamageForTargets(g, playerID, perm.ID(), ab.Effects(), nil)
	outcome := mage.SpellOutcome(ab.Effects())

	var targets []uuid.UUID
	for _, t := range ab.Targets() {
		possible := t.Possible(playerID, perm.Card, g)
		possible = ai.NonRedundantKeywordGrantTargets(g, playerID, perm.ID(), ab.Effects(), possible)
		if len(possible) == 0 {
			return nil, false
		}

		switch t.(type) {
		case *mage.SpellOnStackTarget:
			chosen := bestSpellOnStackTargets(g, playerID, possible, 1)
			if len(chosen) == 0 {
				return nil, false
			}
			targets = append(targets, chosen...)
			continue

		case *mage.DamageAnyTarget:
			if outcome == mage.OutcomeBenefit {
				chosen := bestTargetsForRequirementWithPreference(g, playerID, possible, eval.TargetPump, 0, true, false, hint.PreferTarget)
				if len(chosen) == 0 {
					return nil, false
				}
				targets = append(targets, chosen[0])
				continue
			}
			opponent := g.GetOpponent(playerID)
			if s.weights().TargetFace >= 0.5 && opponent != nil {
				aimedAtFace := false
				for _, id := range possible {
					if id == opponent.PlayerID() && shouldAimBurnAtFace(g, playerID, damage) {
						targets = append(targets, id)
						aimedAtFace = true
						break
					}
				}
				if aimedAtFace {
					continue
				}
			}
			if chosen := bestTargetsForRequirementWithPreference(g, playerID, possible, eval.TargetBurn, damage, false, true, hint.PreferTarget); len(chosen) > 0 {
				targets = append(targets, chosen[0])
				continue
			}
			// A free, repeatable ping (no mana cost, paid only by tapping,
			// e.g. Pirate Ship) always advances the game when aimed at the
			// opponent's face: it costs nothing and chips their life total.
			// Fall back to face even when no creature is worth burning, rather
			// than wasting the activation by passing.
			if opponent != nil && (shouldAimBurnAtFace(g, playerID, damage) || !hasOpponentPermanentTarget(g, playerID, possible) || isFreeRepeatablePing(ab)) {
				aimedAtFace := false
				for _, id := range possible {
					if id == opponent.PlayerID() {
						targets = append(targets, id)
						aimedAtFace = true
						break
					}
				}
				if aimedAtFace {
					continue
				}
			}
			return nil, false

		case *mage.CreatureTarget:
			if outcome == mage.OutcomeBenefit {
				chosen := bestTargetsForRequirementWithPreference(g, playerID, possible, purpose, damage, true, false, hint.PreferTarget)
				if len(chosen) == 0 {
					return nil, false
				}
				targets = append(targets, chosen[0])
				continue
			}
			if chosen := bestTargetsForRequirementWithPreference(g, playerID, possible, purpose, damage, false, true, hint.PreferTarget); len(chosen) > 0 {
				targets = append(targets, chosen[0])
				continue
			}
			return nil, false

		case *mage.PlayerTarget, *mage.OpponentTarget:
			opponent := g.GetOpponent(playerID)
			if opponent == nil {
				return nil, false
			}
			targets = append(targets, opponent.PlayerID())

		default:
			preferOwn := outcome == mage.OutcomeBenefit
			preferOpponent := outcome == mage.OutcomeDetriment
			chosen := bestTargetsForRequirementWithPreference(g, playerID, possible, purpose, damage, preferOwn, preferOpponent, hint.PreferTarget)
			if len(chosen) == 0 {
				if preferOpponent {
					return nil, false
				}
				targets = append(targets, possible[0])
				continue
			}
			targets = append(targets, chosen[0])
		}
	}

	return targets, true
}

func (s *Strategy) Attackers(p mage.Player, g *mage.Game) []uuid.UUID {
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

func (s *Strategy) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	r := combatsolver.SolveDefense(g, p.PlayerID(), combatsolver.Options{Profile: s.solverProfile()})
	return r.Blocks
}

func (s *Strategy) autoSelectTargets(p mage.Player, g *mage.Game, card mage.Card) []uuid.UUID {
	playerID := p.PlayerID()

	var targets []uuid.UUID

	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok || sa.Kind() != mage.ActionSpell {
			continue
		}
		purpose := eval.TargetPurposeForEffects(sa.Effects())
		hint := actionAIHint(sa.Effects(), sa.AIHints())
		if hintedPurpose := eval.TargetPurposeFromAI(hint.TargetPurpose); hintedPurpose != eval.TargetGeneric {
			purpose = hintedPurpose
		}
		damage := spellDamageForTargets(g, playerID, card, nil)
		outcome := mage.SpellOutcome(sa.Effects())

		// Each Target spec is resolved independently and its chosen targets are
		// accumulated, so spells that target multiple things ("up to two target
		// creatures", "target player and target creature") are fully populated
		// rather than only filling the first requirement.
		for _, t := range sa.Targets() {
			possible := t.Possible(playerID, card, g)
			if len(possible) == 0 {
				if t.Min() > 0 {
					return nil
				}
				continue
			}

			chosen := s.selectTargetsForSpec(g, playerID, t, possible, purpose, damage, outcome, hint)
			if len(chosen) < t.Min() {
				return nil
			}
			targets = append(targets, chosen...)
		}
	}

	if len(targets) > 0 {
		return targets
	}

	// Check CastTargets for auras and other cards with cast-time targeting
	for _, t := range card.CastTargets() {
		possible := t.Possible(playerID, card, g)
		if len(possible) == 0 {
			return nil
		}
		switch t.(type) {
		case *mage.CreatureTarget:
			if tg := bestAuraTarget(g, playerID, card, possible); len(tg) > 0 {
				return tg
			}
		default:
			if len(possible) > 0 {
				return []uuid.UUID{possible[0]}
			}
		}
	}

	return targets
}

// selectTargetsForSpec chooses up to t.Max() targets for a single Target spec.
// For specs that accept more than one target it returns the best N (distinct);
// for single-target specs it preserves the original per-type selection logic.
func (s *Strategy) selectTargetsForSpec(g *mage.Game, playerID uuid.UUID, t mage.Target, possible []uuid.UUID, purpose eval.TargetPurpose, damage int, outcome mage.Outcome, hint mage.AIHint) []uuid.UUID {
	n := max(t.Max(), 1)
	opponent := g.GetOpponent(playerID)

	switch t.(type) {
	case *mage.SpellOnStackTarget:
		return bestSpellOnStackTargets(g, playerID, possible, n)

	case *mage.DamageAnyTarget:
		if outcome == mage.OutcomeBenefit {
			return bestNTargetsForRequirement(g, playerID, possible, eval.TargetPump, 0, true, false, hint.PreferTarget, n)
		}
		if n <= 1 {
			return s.selectSingleBurnTarget(g, playerID, possible, damage, hint, opponent)
		}
		chosen := bestNTargetsForRequirement(g, playerID, possible, eval.TargetBurn, damage, false, true, hint.PreferTarget, n)
		if opponent != nil && len(chosen) < n && shouldAimBurnAtFace(g, playerID, damage) &&
			slices.Contains(possible, opponent.PlayerID()) && !slices.Contains(chosen, opponent.PlayerID()) {
			chosen = append(chosen, opponent.PlayerID())
		}
		if len(chosen) == 0 && opponent != nil &&
			(shouldAimBurnAtFace(g, playerID, damage) || !hasOpponentPermanentTarget(g, playerID, possible)) &&
			slices.Contains(possible, opponent.PlayerID()) {
			chosen = []uuid.UUID{opponent.PlayerID()}
		}
		return chosen

	case *mage.CreatureTarget:
		preferOwn := outcome == mage.OutcomeBenefit
		return bestNTargetsForRequirement(g, playerID, possible, purpose, damage, preferOwn, !preferOwn, hint.PreferTarget, n)

	case *mage.OpponentTarget:
		if opponent != nil {
			return []uuid.UUID{opponent.PlayerID()}
		}
		return nil

	case *mage.PlayerTarget:
		// Beneficial player-targeted spells (e.g. Stream of Life's life gain)
		// help us, so point them at ourselves; detrimental ones go at the opponent.
		if outcome == mage.OutcomeBenefit {
			return []uuid.UUID{playerID}
		}
		if opponent != nil {
			return []uuid.UUID{opponent.PlayerID()}
		}
		return []uuid.UUID{playerID}

	default:
		preferOwn := outcome == mage.OutcomeBenefit
		preferOpponent := outcome == mage.OutcomeDetriment
		chosen := bestNTargetsForRequirement(g, playerID, possible, purpose, damage, preferOwn, preferOpponent, hint.PreferTarget, n)
		if len(chosen) == 0 && !preferOpponent && len(possible) > 0 {
			return []uuid.UUID{possible[0]}
		}
		return chosen
	}
}

// selectSingleBurnTarget reproduces the original single-target burn selection:
// aim at the face when that advances the game, otherwise pick the best creature
// to burn, falling back to the face when no creature is worth it.
func (s *Strategy) selectSingleBurnTarget(g *mage.Game, playerID uuid.UUID, possible []uuid.UUID, damage int, hint mage.AIHint, opponent mage.Player) []uuid.UUID {
	if s.weights().TargetFace >= 0.5 && opponent != nil {
		for _, id := range possible {
			if id == opponent.PlayerID() && shouldAimBurnAtFace(g, playerID, damage) {
				return []uuid.UUID{id}
			}
		}
	}
	if hint.PreferTarget == mage.PreferOpponentFaceIfLethal && opponent != nil && damage >= opponent.Life() {
		return []uuid.UUID{opponent.PlayerID()}
	}
	if tg := bestTargetsForRequirementWithPreference(g, playerID, possible, eval.TargetBurn, damage, false, true, hint.PreferTarget); len(tg) > 0 {
		return tg
	}
	if opponent != nil && (shouldAimBurnAtFace(g, playerID, damage) || !hasOpponentPermanentTarget(g, playerID, possible)) {
		for _, id := range possible {
			if id == opponent.PlayerID() {
				return []uuid.UUID{id}
			}
		}
	}
	return nil
}

// canCastWhileReserving reports whether the player can pay for `cast` and still
// have enough mana left to cast `reserve` afterward. It checks affordability of
// the two costs combined, so the mana solver honors colors, dual lands, and
// conversions rather than a color-blind mana count.
func canCastWhileReserving(g *mage.Game, playerID uuid.UUID, cast, reserve mage.Card) bool {
	combined := combinedManaCost(cast.ManaCost(), reserve.ManaCost())
	return g.CanAfford(playerID, combined, mage.SpellContextForCard(cast))
}

// combinedManaCost sums two mana costs. The {X} portion is ignored: X-cost
// spells consume all remaining mana, so callers exclude them from reservation.
func combinedManaCost(a, b core.ManaCost) core.ManaCost {
	a.Generic += b.Generic
	a.White += b.White
	a.Blue += b.Blue
	a.Black += b.Black
	a.Red += b.Red
	a.Green += b.Green
	a.Hybrid = append(append([]core.HybridSymbol{}, a.Hybrid...), b.Hybrid...)
	a.HasX = false
	a.XCount = 0
	return a
}

func hasOpponentPermanentTarget(g *mage.Game, playerID uuid.UUID, possible []uuid.UUID) bool {
	for _, id := range possible {
		if perm := g.FindPermanent(id); perm != nil && perm.Controller != playerID {
			return true
		}
	}
	return false
}

func requiresTargets(card mage.Card) bool {
	if len(card.CastTargets()) > 0 {
		return true
	}
	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if ok && sa.Kind() == mage.ActionSpell && len(sa.Targets()) > 0 {
			return true
		}
	}
	return false
}

// auraAIProfile retrieves an aura's AI targeting profile, if the card declares one.
func auraAIProfile(card mage.Card) (mage.AuraAIProfile, bool) {
	if profiler, ok := card.(interface {
		AuraAIProfile() (mage.AuraAIProfile, bool)
	}); ok {
		return profiler.AuraAIProfile()
	}
	return mage.AuraAIProfile{}, false
}

// bestAuraTarget chooses the enchant target for an aura. Beneficial auras go on
// our own best creature; detrimental or control-stealing auras (Weakness,
// Control Magic) go on the opponent's best creature. Auras that raise power but
// lower toughness (Immolation, +2/-2) only help us when they kill the creature,
// so they only target an opponent creature whose toughness they would reduce to
// zero — otherwise they would buff the opponent.
func bestAuraTarget(g *mage.Game, playerID uuid.UUID, card mage.Card, possible []uuid.UUID) []uuid.UUID {
	profile, ok := auraAIProfile(card)
	detrimental := ok && (profile.StealsControl || profile.PowerBoost < 0 || profile.ToughnessBoost < 0)
	if !detrimental {
		return bestTargetsForRequirement(g, playerID, possible, eval.TargetAura, 0, true, false)
	}

	onlyIfLethal := profile.ToughnessBoost < 0 && profile.PowerBoost > 0

	bestID := uuid.Nil
	bestScore := -10000
	for _, id := range possible {
		perm := g.FindPermanent(id)
		if perm == nil || perm.Controller == playerID || !perm.HasType(core.TypeCreature) {
			continue
		}
		if onlyIfLethal && perm.CurrentToughness(g)+profile.ToughnessBoost-perm.Damage > 0 {
			continue
		}
		score := eval.PermanentValueForTargeting(g, perm, eval.TargetRemoval)
		if score > bestScore {
			bestScore = score
			bestID = id
		}
	}
	if bestID == uuid.Nil {
		return nil
	}
	return []uuid.UUID{bestID}
}

// bestSpellOnStackTargets chooses which spell(s) on the stack to counter. It
// only ever targets an opponent's spell (never our own) and prefers the
// highest-value one, estimated by SpellValue from the caster's perspective so
// the most threatening spell is countered first. `possible` holds the SourceIDs
// of legal spell targets as returned by SpellOnStackTarget.Possible.
func bestSpellOnStackTargets(g *mage.Game, playerID uuid.UUID, possible []uuid.UUID, n int) []uuid.UUID {
	if n < 1 {
		n = 1
	}
	type scoredSpell struct {
		id    uuid.UUID
		score int
	}
	var ranked []scoredSpell
	for _, obj := range g.StackObjects() {
		if obj.IsAbility || obj.Card == nil || obj.Controller == playerID {
			continue
		}
		if !slices.Contains(possible, obj.SourceID) {
			continue
		}
		score := max(
			// Any opposing spell legal to counter is worth at least a floor value, so
			// a low-cost spell is still a valid target rather than being skipped.
			eval.SpellValue(obj.Card, g.GetPlayer(obj.Controller), g), 1)
		ranked = append(ranked, scoredSpell{obj.SourceID, score})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})
	if len(ranked) > n {
		ranked = ranked[:n]
	}
	out := make([]uuid.UUID, len(ranked))
	for i := range ranked {
		out[i] = ranked[i].id
	}
	return out
}

func bestTargetsForRequirement(g *mage.Game, playerID uuid.UUID, possible []uuid.UUID, purpose eval.TargetPurpose, damage int, preferOwn, preferOpponent bool) []uuid.UUID {
	return bestTargetsForRequirementWithPreference(g, playerID, possible, purpose, damage, preferOwn, preferOpponent, mage.PreferNoTarget)
}

// bestNTargetsForRequirement returns up to n distinct targets (best first) that
// each score as worthwhile, for target specs that accept more than one target.
// For n <= 1 it delegates to the single-target selector so existing behavior is
// preserved exactly.
func bestNTargetsForRequirement(g *mage.Game, playerID uuid.UUID, possible []uuid.UUID, purpose eval.TargetPurpose, damage int, preferOwn, preferOpponent bool, preference mage.AITargetPreference, n int) []uuid.UUID {
	if n <= 1 {
		return bestTargetsForRequirementWithPreference(g, playerID, possible, purpose, damage, preferOwn, preferOpponent, preference)
	}

	type scoredTarget struct {
		id    uuid.UUID
		score int
	}
	var scored []scoredTarget
	for _, id := range possible {
		perm := g.FindPermanent(id)
		if perm == nil {
			continue
		}
		if preferOwn && perm.Controller != playerID {
			continue
		}
		if preferOpponent && perm.Controller == playerID {
			continue
		}
		score := eval.TargetValueForPurpose(g, playerID, id, purpose, damage)
		score += targetPreferenceBonus(g, playerID, perm, preference, damage)
		if score < 1 {
			continue
		}
		scored = append(scored, scoredTarget{id, score})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})
	if len(scored) > n {
		scored = scored[:n]
	}
	out := make([]uuid.UUID, len(scored))
	for i := range scored {
		out[i] = scored[i].id
	}
	return out
}

func bestTargetsForRequirementWithPreference(g *mage.Game, playerID uuid.UUID, possible []uuid.UUID, purpose eval.TargetPurpose, damage int, preferOwn, preferOpponent bool, preference mage.AITargetPreference) []uuid.UUID {
	if preference == mage.PreferSmallestOwnCreature {
		if target := smallestOwnCreatureTarget(g, playerID, possible); target != uuid.Nil {
			return []uuid.UUID{target}
		}
	}

	bestID := uuid.Nil
	bestScore := -10000
	for _, id := range possible {
		perm := g.FindPermanent(id)
		if perm == nil {
			continue
		}
		if preferOwn && perm.Controller != playerID {
			continue
		}
		if preferOpponent && perm.Controller == playerID {
			continue
		}
		score := eval.TargetValueForPurpose(g, playerID, id, purpose, damage)
		score += targetPreferenceBonus(g, playerID, perm, preference, damage)
		if score > bestScore {
			bestScore = score
			bestID = id
		}
	}
	if bestID == uuid.Nil || bestScore < 1 {
		return nil
	}
	return []uuid.UUID{bestID}
}

func smallestOwnCreatureTarget(g *mage.Game, playerID uuid.UUID, possible []uuid.UUID) uuid.UUID {
	bestID := uuid.Nil
	bestScore := 1 << 30
	for _, id := range possible {
		perm := g.FindPermanent(id)
		if perm == nil || perm.Controller != playerID || !perm.HasType(core.TypeCreature) {
			continue
		}
		score := eval.PermanentValueForTargeting(g, perm, eval.TargetGeneric)
		if score < bestScore {
			bestScore = score
			bestID = id
		}
	}
	return bestID
}

func targetPreferenceBonus(g *mage.Game, playerID uuid.UUID, perm *mage.Permanent, preference mage.AITargetPreference, damage int) int {
	switch preference {
	case mage.PreferOpponentCreature:
		if perm.Controller != playerID && perm.HasType(core.TypeCreature) {
			return 40
		}
	case mage.PreferOwnCreature:
		if perm.Controller == playerID && perm.HasType(core.TypeCreature) {
			return 40
		}
	case mage.PreferLethalCreature:
		if perm.Controller != playerID && perm.HasType(core.TypeCreature) && damage >= perm.CurrentToughness(g)-perm.Damage {
			return 50
		}
	case mage.PreferEvasiveCreature:
		if perm.HasType(core.TypeCreature) && hasAITargetingEvasion(perm) {
			return 35
		}
	case mage.PreferLargestThreat:
		return eval.PermanentValueForTargeting(g, perm, eval.TargetRemoval)
	case mage.PreferSmallestOwnCreature:
		if perm.Controller == playerID && perm.HasType(core.TypeCreature) {
			return 1000 - eval.PermanentValueForTargeting(g, perm, eval.TargetGeneric)
		}
		return -100
	}
	return 0
}

func hasAITargetingEvasion(perm *mage.Permanent) bool {
	return perm.HasKeyword(core.UnblockableKW) ||
		perm.HasKeyword(core.Flying) ||
		perm.HasKeyword(core.Fear) ||
		perm.HasKeyword(core.Menace) ||
		perm.HasKeyword(core.Islandwalk) ||
		perm.HasKeyword(core.Swampwalk) ||
		perm.HasKeyword(core.Forestwalk) ||
		perm.HasKeyword(core.Mountainwalk) ||
		perm.HasKeyword(core.Plainswalk)
}

func shouldAimBurnAtFace(g *mage.Game, playerID uuid.UUID, damage int) bool {
	opponent := g.GetOpponent(playerID)
	if opponent == nil || damage <= 0 {
		return false
	}
	if damage >= opponent.Life() {
		return true
	}
	lethal := eval.CalculateLethal(g, playerID)
	if lethal.MyBoardDamage+damage >= opponent.Life() {
		return true
	}
	race := eval.CalculateRace(g, playerID)
	return race.Racing && race.MyClock <= race.TheirClock && opponent.Life() <= 10
}

func spellDamageForTargets(g *mage.Game, playerID uuid.UUID, card mage.Card, targets []uuid.UUID) int {
	damage := 0
	for _, ab := range card.Abilities() {
		if sa, ok := ab.(*mage.SpellAbility); ok && sa.Kind() == mage.ActionSpell {
			for _, e := range sa.Effects() {
				if dv := e.Properties().DamageValue; dv != nil {
					damage = dv.Resolve(g, card.ID(), playerID, targets)
				}
			}
		}
	}
	return damage
}

func cardAIHint(card mage.Card) mage.AIHint {
	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok || sa.Kind() != mage.ActionSpell {
			continue
		}
		return actionAIHint(sa.Effects(), sa.AIHints())
	}
	return mage.AIHint{}
}

func abilityAIHint(ab mage.ActivatedAbility) mage.AIHint {
	if ad, ok := ab.(*mage.ActionDefinition); ok {
		return actionAIHint(ad.Effects(), ad.AIHints())
	}
	return actionAIHint(ab.Effects(), nil)
}

func actionAIHint(effects []mage.Effect, hints []mage.AIHint) mage.AIHint {
	var out mage.AIHint
	for _, e := range effects {
		props := e.Properties()
		mergeAIHint(&out, mage.AIHint{
			Roles:         props.AIRoles,
			Timing:        props.Timing,
			TargetPurpose: props.TargetPurposeOverride,
			PreferTarget:  props.PreferTarget,
			ValueBias:     props.ValueBias,
		})
	}
	for _, hint := range hints {
		mergeAIHint(&out, hint)
	}
	return out
}

func mergeAIHint(dst *mage.AIHint, src mage.AIHint) {
	if len(src.Roles) > 0 {
		dst.Roles = append(dst.Roles, src.Roles...)
	}
	if src.Timing != mage.AITimingAny {
		dst.Timing = src.Timing
	}
	if src.TargetPurpose != mage.AITargetGeneric {
		dst.TargetPurpose = src.TargetPurpose
	}
	if src.PreferTarget != mage.PreferNoTarget {
		dst.PreferTarget = src.PreferTarget
	}
	dst.ValueBias += src.ValueBias
}

func aiHintAllowsTiming(hint mage.AIHint, g *mage.Game, mainPhase bool) bool {
	switch hint.Timing {
	case mage.AITimingMainPhase:
		return mainPhase
	case mage.AITimingPostCombat:
		return g.GetStep() == core.PostcombatMain
	case mage.AITimingCombatOnly:
		return g.GetStep() == core.DeclareAttackers || g.GetStep() == core.DeclareBlockers ||
			g.GetStep() == core.CombatDamage || g.GetStep() == core.FirstStrikeDamage
	case mage.AITimingResponseOnly:
		return len(g.StackObjects()) > 0
	case mage.AITimingEndStep:
		return g.GetStep() == core.EndStep
	default:
		return true
	}
}

// isFreeRepeatablePing reports whether an activated ability deals damage and
// costs no mana (paid only by tapping or other non-mana costs). Such pings can
// be used every turn at no resource cost, so aiming them at the opponent's face
// is always a small free gain even when no creature is worth burning.
func isFreeRepeatablePing(ab mage.ActivatedAbility) bool {
	dealsDamage := false
	for _, e := range ab.Effects() {
		if e.Properties().DamageValue != nil && e.Properties().Outcome == mage.OutcomeDetriment {
			dealsDamage = true
			break
		}
	}
	if !dealsDamage {
		return false
	}
	for _, c := range ab.Costs() {
		if mcp, ok := c.(*mage.ManaCostPayment); ok && mcp.MC.CMC() > 0 {
			return false
		}
	}
	return true
}

func abilityDamageForTargets(g *mage.Game, playerID, sourceID uuid.UUID, effects []mage.Effect, targets []uuid.UUID) int {
	damage := 0
	for _, e := range effects {
		if dv := e.Properties().DamageValue; dv != nil {
			damage = dv.Resolve(g, sourceID, playerID, targets)
		}
	}
	return damage
}

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
		// If targeting a creature, pay just enough to kill it: lethal damage
		// is its current toughness minus damage already marked.
		if perm := g.FindPermanent(targets[0]); perm != nil && perm.Controller != playerID {
			lethal := perm.CurrentToughness(g) - perm.Damage
			if lethal > 0 && lethal <= maxX {
				return lethal
			}
		}
	}

	if isDetrimentalTapXSpell(card) {
		opponent := g.GetOpponent(playerID)
		if opponent == nil {
			return 0
		}
		count := 0
		for _, perm := range g.AllBattlefield() {
			if perm.Controller == opponent.PlayerID() && perm.HasType(core.TypeCreature) && !perm.Tapped {
				count++
			}
		}
		if count == 0 {
			return 0
		}
		return min(maxX, count)
	}

	// Default: spend all available mana.
	return maxX
}

func isDetrimentalTapXSpell(card mage.Card) bool {
	for _, a := range card.Abilities() {
		if sa, ok := a.(*mage.SpellAbility); ok && sa.Kind() == mage.ActionSpell {
			for _, e := range sa.Effects() {
				props := e.Properties()
				if props.Taps && props.Outcome == mage.OutcomeDetriment {
					return true
				}
			}
		}
	}
	return false
}

// chooseBestLand selects the best land to play from hand. It prefers lands that
// make the highest-value currently stranded spells castable.
func chooseBestLand(p mage.Player, g *mage.Game) mage.Card {
	hand := p.Hand()

	var lands []mage.Card
	neededColors := make(map[core.Color]int)
	baseColors := availableManaColors(g, p.PlayerID())
	baseTotal := eval.CountAvailableMana(g, p.PlayerID())

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
		landColors, anyColor := landManaColors(land)
		nextColors := copyColorCounts(baseColors)
		if anyColor {
			for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green} {
				nextColors[color]++
			}
		}
		for _, color := range landColors {
			nextColors[color]++
		}
		nextTotal := baseTotal + 1

		for _, spell := range hand {
			if spell.HasType(core.TypeLand) {
				continue
			}
			if manaCanCast(spell.ManaCost(), baseColors, baseTotal) {
				continue
			}
			if manaCanCast(spell.ManaCost(), nextColors, nextTotal) {
				score += eval.SpellValue(spell, p, g) * 4
			} else if coloredManaProgress(spell.ManaCost(), nextColors) > coloredManaProgress(spell.ManaCost(), baseColors) {
				score += eval.SpellValue(spell, p, g)
			}
		}

		for _, color := range landColors {
			score += neededColors[color] * 2
		}
		if anyColor {
			for _, need := range neededColors {
				score += need
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

func availableManaColors(g *mage.Game, playerID uuid.UUID) map[core.Color]int {
	colors := make(map[core.Color]int)
	for _, perm := range g.FilterBattlefield(mage.And(mage.ControlledBy(playerID), mage.IsUntapped)) {
		for _, a := range perm.RuntimeAbilities {
			ma, ok := mage.UnwrapAbility(a).(*mage.ManaAbility)
			if !ok {
				continue
			}
			if ma.HasAnyColor() {
				for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green} {
					colors[color]++
				}
			} else if ma.PrimaryColor() != core.Colorless {
				colors[ma.PrimaryColor()]++
			}
		}
	}
	return colors
}

func landManaColors(card mage.Card) ([]core.Color, bool) {
	var colors []core.Color
	anyColor := false
	for _, a := range card.Abilities() {
		ma, ok := mage.UnwrapAbility(a).(*mage.ManaAbility)
		if !ok {
			continue
		}
		if ma.HasAnyColor() {
			anyColor = true
			continue
		}
		if ma.PrimaryColor() != core.Colorless {
			colors = append(colors, ma.PrimaryColor())
		}
	}
	return colors, anyColor
}

func copyColorCounts(src map[core.Color]int) map[core.Color]int {
	dst := make(map[core.Color]int, len(src))
	maps.Copy(dst, src)
	return dst
}

func manaCanCast(mc core.ManaCost, colors map[core.Color]int, total int) bool {
	required := mc.White + mc.Blue + mc.Black + mc.Red + mc.Green
	if colors[core.White] < mc.White ||
		colors[core.Blue] < mc.Blue ||
		colors[core.Black] < mc.Black ||
		colors[core.Red] < mc.Red ||
		colors[core.Green] < mc.Green {
		return false
	}
	return total >= required+mc.Generic
}

func coloredManaProgress(mc core.ManaCost, colors map[core.Color]int) int {
	score := 0
	score += min(colors[core.White], mc.White)
	score += min(colors[core.Blue], mc.Blue)
	score += min(colors[core.Black], mc.Black)
	score += min(colors[core.Red], mc.Red)
	score += min(colors[core.Green], mc.Green)
	return score
}
