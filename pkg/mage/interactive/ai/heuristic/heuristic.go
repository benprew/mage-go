// Package heuristic implements a fast, personality-driven AI strategy that
// makes decisions via local heuristics (without search).
package heuristic

import (
	"maps"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/ai"
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
	return combatsolver.Profile{
		Weights:        w.Weights,
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
			if requiresTargets(bestCard) && len(targets) == 0 {
				return interactive.PriorityAction{Type: interactive.ActionPass}
			}
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
			if !g.CanAfford(playerID, card.ManaCost(), mage.SpellContextForCard(card)) {
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

		targets, ok := s.autoSelectAbilityTargets(p, g, perm, ab)
		if !ok {
			return nil
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

func (s *Strategy) autoSelectAbilityTargets(p mage.Player, g *mage.Game, perm *mage.Permanent, ab mage.ActivatedAbility) ([]uuid.UUID, bool) {
	playerID := p.PlayerID()
	purpose := eval.TargetPurposeForEffects(ab.Effects())
	damage := abilityDamageForTargets(g, playerID, perm.ID(), ab.Effects(), nil)
	outcome := mage.SpellOutcome(ab.Effects())

	var targets []uuid.UUID
	for _, t := range ab.Targets() {
		possible := t.Possible(playerID, perm.Card, g)
		if len(possible) == 0 {
			return nil, false
		}

		switch t.(type) {
		case *mage.DamageAnyTarget:
			if outcome == mage.OutcomeBenefit {
				chosen := bestTargetsForRequirement(g, playerID, possible, eval.TargetPump, 0, true, false)
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
			if chosen := bestTargetsForRequirement(g, playerID, possible, eval.TargetBurn, damage, false, true); len(chosen) > 0 {
				targets = append(targets, chosen[0])
				continue
			}
			if opponent != nil && (shouldAimBurnAtFace(g, playerID, damage) || !hasOpponentPermanentTarget(g, playerID, possible)) {
				targets = append(targets, opponent.PlayerID())
				continue
			}
			return nil, false

		case *mage.CreatureTarget:
			if outcome == mage.OutcomeBenefit {
				chosen := bestTargetsForRequirement(g, playerID, possible, purpose, damage, true, false)
				if len(chosen) == 0 {
					return nil, false
				}
				targets = append(targets, chosen[0])
				continue
			}
			if chosen := bestTargetsForRequirement(g, playerID, possible, purpose, damage, false, true); len(chosen) > 0 {
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
			chosen := bestTargetsForRequirement(g, playerID, possible, purpose, damage, preferOwn, preferOpponent)
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

	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok || sa.Kind() != mage.ActionSpell {
			continue
		}
		purpose := eval.TargetPurposeForEffects(sa.Effects())
		damage := spellDamageForTargets(g, playerID, card, nil)
		for _, t := range sa.Targets() {
			possible := t.Possible(playerID, card, g)
			if len(possible) == 0 {
				return nil
			}

			outcome := mage.SpellOutcome(sa.Effects())
			switch t.(type) {
			case *mage.DamageAnyTarget:
				if outcome == mage.OutcomeBenefit {
					return bestTargetsForRequirement(g, playerID, possible, eval.TargetPump, 0, true, false)
				}
				opponent := g.GetOpponent(playerID)
				if s.weights().TargetFace >= 0.5 && opponent != nil {
					for _, id := range possible {
						if id == opponent.PlayerID() && shouldAimBurnAtFace(g, playerID, damage) {
							return []uuid.UUID{id}
						}
					}
				}
				if targets := bestTargetsForRequirement(g, playerID, possible, eval.TargetBurn, damage, false, true); len(targets) > 0 {
					return targets
				}
				if opponent != nil && (shouldAimBurnAtFace(g, playerID, damage) || !hasOpponentPermanentTarget(g, playerID, possible)) {
					return []uuid.UUID{opponent.PlayerID()}
				}

			case *mage.CreatureTarget:
				if outcome == mage.OutcomeBenefit {
					return bestTargetsForRequirement(g, playerID, possible, purpose, damage, true, false)
				}
				if targets := bestTargetsForRequirement(g, playerID, possible, purpose, damage, false, true); len(targets) > 0 {
					return targets
				}
				// Only fall back to own creatures for beneficial spells;
				// never target your own creature with removal.
				if outcome == mage.OutcomeBenefit {
					return bestTargetsForRequirement(g, playerID, possible, purpose, damage, true, false)
				}

			case *mage.PlayerTarget, *mage.OpponentTarget:
				opponent := g.GetOpponent(playerID)
				if opponent != nil {
					return []uuid.UUID{opponent.PlayerID()}
				}

			default:
				preferOwn := outcome == mage.OutcomeBenefit
				preferOpponent := outcome == mage.OutcomeDetriment
				if targets := bestTargetsForRequirement(g, playerID, possible, purpose, damage, preferOwn, preferOpponent); len(targets) > 0 {
					return targets
				}
				if !preferOpponent && len(possible) > 0 {
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
			if targets := bestTargetsForRequirement(g, playerID, possible, eval.TargetAura, 0, true, false); len(targets) > 0 {
				return targets
			}
		default:
			if len(possible) > 0 {
				return []uuid.UUID{possible[0]}
			}
		}
	}

	return nil
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

func bestTargetsForRequirement(g *mage.Game, playerID uuid.UUID, possible []uuid.UUID, purpose eval.TargetPurpose, damage int, preferOwn, preferOpponent bool) []uuid.UUID {
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
