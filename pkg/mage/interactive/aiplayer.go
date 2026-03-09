package interactive

import (
	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// ─── AI Strategy ───────────────────────────────────────────────────────────

// SpellOrder controls the order in which the AI evaluates castable spells.
type SpellOrder int

const (
	MostExpensiveFirst SpellOrder = iota
	CheapestFirst
)

// Personality parameterises how an AIPlayer makes decisions.
// Deprecated: Use WeightedPersonality for new code. Personality is retained
// for backward compatibility and can be converted via ToWeighted().
type Personality struct {
	Name                string
	CastOrder           SpellOrder
	HoldInstants        bool // if true, only cast instants in response to opponent actions
	AttackAll           bool // if true, attack with every eligible creature; if false, only attack when profitable
	BlockPowerThreshold int  // block attackers with power >= this (0 = block everything)
	TargetFace          bool // if true, damage spells target the opponent player rather than creatures
}

// ToWeighted converts a boolean Personality into a continuous WeightedPersonality.
func (p Personality) ToWeighted() WeightedPersonality {
	wp := WeightedPersonality{Name: p.Name}

	// Map boolean AttackAll to Aggression
	if p.AttackAll {
		wp.Aggression = 1.0
	} else {
		wp.Aggression = 0.0
	}

	// Map BlockPowerThreshold to BlockThreshold (0 → 0.0, 99 → ~1.0)
	// Threshold 0 means block everything (0.0), higher means block less.
	if p.BlockPowerThreshold >= 99 {
		wp.BlockThreshold = 0.95
	} else {
		wp.BlockThreshold = float64(p.BlockPowerThreshold) / 10.0
		if wp.BlockThreshold > 1.0 {
			wp.BlockThreshold = 1.0
		}
	}

	// Map HoldInstants boolean
	if p.HoldInstants {
		wp.HoldInstants = 1.0
	} else {
		wp.HoldInstants = 0.0
	}

	// Map TargetFace boolean
	if p.TargetFace {
		wp.TargetFace = 1.0
	} else {
		wp.TargetFace = 0.0
	}

	// Map CastOrder to CurvePreference
	if p.CastOrder == MostExpensiveFirst {
		wp.CurvePreference = 1.0
	} else {
		wp.CurvePreference = 0.0
	}

	// Default evaluation weights (neutral)
	wp.LifeWeight = 2.0
	wp.BoardWeight = 2.0
	wp.CardWeight = 2.0
	wp.ManaWeight = 1.0
	wp.TempoWeight = 1.0

	return wp
}

// WeightedPersonality provides continuous-valued weights for AI decision-making.
// Evaluation weights flow into the state evaluator. Decision weights (0.0 to 1.0)
// control combat, targeting, and spell-casting preferences.
type WeightedPersonality struct {
	Name string

	// Evaluation weights (flow into StateEvaluator)
	LifeWeight  float64 // importance of life difference
	BoardWeight float64 // importance of creature board
	CardWeight  float64 // importance of hand advantage
	ManaWeight  float64 // importance of mana development
	TempoWeight float64 // importance of untapped mana

	// Decision weights (0.0 to 1.0 continuous)
	Aggression      float64 // 1.0 = attack everything, 0.0 = only profitable
	BlockThreshold  float64 // 0.0 = block everything, 1.0 = never block
	HoldInstants    float64 // 0.0 = cast immediately, 1.0 = always hold
	TargetFace      float64 // 0.0 = always target creatures, 1.0 = always go face
	CurvePreference float64 // 0.0 = cheapest first, 1.0 = most expensive first
}

// Preset weighted personalities.
var (
	AggroWeighted = WeightedPersonality{
		Name:            "Aggro",
		LifeWeight:      1.0,
		BoardWeight:     3.0,
		CardWeight:      0.5,
		ManaWeight:      0.5,
		TempoWeight:     0.5,
		Aggression:      1.0,
		BlockThreshold:  0.7,
		HoldInstants:    0.0,
		TargetFace:      0.3,
		CurvePreference: 0.0,
	}
	ControlWeighted = WeightedPersonality{
		Name:            "Control",
		LifeWeight:      4.0,
		BoardWeight:     1.0,
		CardWeight:      3.0,
		ManaWeight:      1.0,
		TempoWeight:     2.0,
		Aggression:      0.0,
		BlockThreshold:  0.0,
		HoldInstants:    1.0,
		TargetFace:      0.0,
		CurvePreference: 1.0,
	}
	MidrangeWeighted = WeightedPersonality{
		Name:            "Midrange",
		LifeWeight:      2.0,
		BoardWeight:     3.0,
		CardWeight:      2.0,
		ManaWeight:      1.0,
		TempoWeight:     1.0,
		Aggression:      0.7,
		BlockThreshold:  0.3,
		HoldInstants:    0.0,
		TargetFace:      0.0,
		CurvePreference: 1.0,
	}
	TempoWeighted = WeightedPersonality{
		Name:            "Tempo",
		LifeWeight:      1.5,
		BoardWeight:     2.0,
		CardWeight:      1.5,
		ManaWeight:      2.0,
		TempoWeight:     3.0,
		Aggression:      0.8,
		BlockThreshold:  0.3,
		HoldInstants:    0.7,
		TargetFace:      0.0,
		CurvePreference: 0.0,
	}
	BurnWeighted = WeightedPersonality{
		Name:            "Burn",
		LifeWeight:      0.5,
		BoardWeight:     1.0,
		CardWeight:      0.5,
		ManaWeight:      0.5,
		TempoWeight:     0.5,
		Aggression:      1.0,
		BlockThreshold:  0.95,
		HoldInstants:    0.0,
		TargetFace:      1.0,
		CurvePreference: 0.0,
	}
)

// Preset personalities (boolean, deprecated — use weighted presets for new code).
var (
	AggroPersonality = Personality{
		Name:                "Aggro",
		CastOrder:           CheapestFirst,
		HoldInstants:        false,
		AttackAll:           true,
		BlockPowerThreshold: 3,
	}
	ControlPersonality = Personality{
		Name:                "Control",
		CastOrder:           MostExpensiveFirst,
		HoldInstants:        true,
		AttackAll:           false,
		BlockPowerThreshold: 0,
	}
	MidrangePersonality = Personality{
		Name:                "Midrange",
		CastOrder:           MostExpensiveFirst,
		HoldInstants:        false,
		AttackAll:           true,
		BlockPowerThreshold: 3,
	}
	TempoPersonality = Personality{
		Name:                "Tempo",
		CastOrder:           CheapestFirst,
		HoldInstants:        true,
		AttackAll:           true,
		BlockPowerThreshold: 2,
	}
	// BurnPersonality represents a dedicated burn deck: race to the face,
	// never block, always target the opponent player with damage spells.
	BurnPersonality = Personality{
		Name:                "Burn",
		CastOrder:           CheapestFirst,
		HoldInstants:        false,
		AttackAll:           true,
		BlockPowerThreshold: 99, // effectively never blocks
		TargetFace:          true,
	}
)

// AIStrategy is the decision-making interface for computer-controlled players.
type AIStrategy interface {
	PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) PriorityAction
	Attackers(p mage.Player, g *mage.Game) []uuid.UUID
	Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment
}

// HeuristicStrategy implements AIStrategy using personality-driven heuristics.
// Internally it uses WeightedPersonality for all decisions.
type HeuristicStrategy struct {
	Personality Personality         // retained for backward compatibility
	Weights     WeightedPersonality // used for all decision-making
	weightsInit bool               // true if Weights was explicitly set
}

// NewHeuristicStrategy creates a HeuristicStrategy from a WeightedPersonality.
func NewHeuristicStrategy(w WeightedPersonality) *HeuristicStrategy {
	return &HeuristicStrategy{Weights: w, weightsInit: true}
}

// newHeuristicFromOld creates a HeuristicStrategy from a boolean Personality,
// converting it to weighted form internally.
func newHeuristicFromOld(p Personality) *HeuristicStrategy {
	return &HeuristicStrategy{Personality: p, Weights: p.ToWeighted(), weightsInit: true}
}

// weights returns the effective WeightedPersonality. If Weights was not
// explicitly set (e.g. struct literal with only Personality), it auto-converts
// from the boolean Personality for backward compatibility.
func (s *HeuristicStrategy) weights() WeightedPersonality {
	if !s.weightsInit {
		s.Weights = s.Personality.ToWeighted()
		s.weightsInit = true
	}
	return s.Weights
}

func (s *HeuristicStrategy) PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) PriorityAction {
	playerID := p.PlayerID()

	// Phase 0A: If opponent has lethal and we have removal, prioritize it.
	lethal := CalculateLethal(g, playerID)
	if lethal.TheyHaveLethal {
		if removal := s.findBestRemoval(p, g); removal != nil {
			return *removal
		}
	}

	if mainPhase {
		if landsPlayed < 1 {
			for _, c := range p.Hand() {
				if c.HasType(core.TypeLand) {
					return PriorityAction{
						Type:     ActionPlayLand,
						CardID:   c.ID(),
						CardName: c.Name(),
					}
				}
			}
		}

		// Phase 0C: Compute available mana and hand CMCs for curve awareness.
		availMana := countAvailableMana(g, playerID)
		cmcs := handCMCs(p.Hand())

		var bestCard mage.Card
		bestScore := -1
		for _, card := range g.GetCastableSpells(playerID) {
			if card.HasType(core.TypeInstant) {
				continue
			}
			score := spellValue(card, p, g)
			// Phase 0C: Add mana curve bonus
			curveBonus := manaCurveBonus(card.ManaCost().CMC(), availMana, cmcs)
			score += int(curveBonus)
			if score > bestScore {
				bestScore = score
				bestCard = card
			}
		}
		if bestCard != nil && !spellIsWorthless(bestCard, p, g) {
			targets := s.autoSelectTargets(p, g, bestCard)
			return PriorityAction{
				Type:     ActionCastSpell,
				CardID:   bestCard.ID(),
				CardName: bestCard.Name(),
				Targets:  targets,
			}
		}
	}

	// Phase 0D: Consider activating non-mana abilities.
	if action := s.considerAbilityActivation(p, g); action != nil {
		return *action
	}

	// HoldInstants: at 1.0 always hold during main phase; at 0.0 never hold.
	holdNow := mainPhase && s.weights().HoldInstants > 0.5
	if !holdNow {
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
			if hasUsableEffect {
				targets := s.autoSelectTargets(p, g, card)
				if len(targets) > 0 {
					return PriorityAction{
						Type:     ActionCastSpell,
						CardID:   card.ID(),
						CardName: card.Name(),
						Targets:  targets,
					}
				}
			}
		}
	}

	return PriorityAction{Type: ActionPass}
}

// findBestRemoval searches for the best removal spell in hand that can kill
// the opponent's biggest threat. Used when opponent has lethal on board.
func (s *HeuristicStrategy) findBestRemoval(p mage.Player, g *mage.Game) *PriorityAction {
	playerID := p.PlayerID()
	opponent := g.GetOpponent(playerID)
	if opponent == nil {
		return nil
	}

	for _, card := range p.Hand() {
		if !g.CanAfford(playerID, card.ManaCost()) {
			continue
		}
		// Check if this spell has a detrimental effect (removal)
		for _, a := range card.Abilities() {
			sa, ok := a.(*mage.SpellAbility)
			if !ok {
				continue
			}
			outcome := mage.SpellOutcome(sa.Effects())
			if outcome == mage.OutcomeDetriment {
				targets := s.autoSelectTargets(p, g, card)
				if len(targets) > 0 {
					// Verify we're targeting an opponent's permanent
					for _, tid := range targets {
						perm := g.FindPermanent(tid)
						if perm != nil && perm.Controller == opponent.PlayerID() {
							return &PriorityAction{
								Type:     ActionCastSpell,
								CardID:   card.ID(),
								CardName: card.Name(),
								Targets:  targets,
							}
						}
					}
				}
			}
		}
	}
	return nil
}

// considerAbilityActivation checks for valuable non-mana activated abilities
// and returns an action to activate the best one. Phase 0D.
func (s *HeuristicStrategy) considerAbilityActivation(p mage.Player, g *mage.Game) *PriorityAction {
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
		// Score the ability by its effects
		if info.AbilityIndex < 0 || info.AbilityIndex >= len(perm.RuntimeAbilities) {
			continue
		}
		ab, ok := mage.UnwrapAbility(perm.RuntimeAbilities[info.AbilityIndex]).(mage.ActivatedAbility)
		if !ok {
			continue
		}
		score := abilityQuality(ab)
		if score > bestScore {
			bestScore = score
			bestInfo = info
		}
	}

	// Only activate if the ability is reasonably valuable (score >= 3)
	if bestInfo != nil && bestScore >= 3 {
		perm := g.FindPermanent(bestInfo.PermanentID)
		if perm == nil {
			return nil
		}
		ab, ok := mage.UnwrapAbility(perm.RuntimeAbilities[bestInfo.AbilityIndex]).(mage.ActivatedAbility)
		if !ok {
			return nil
		}

		// Auto-select targets for the ability
		var targets []uuid.UUID
		for _, t := range ab.Targets() {
			possible := t.Possible(playerID, perm.Card, g)
			if len(possible) == 0 {
				return nil // can't activate, no valid targets
			}
			// Simple target selection: pick best opponent creature or opponent player
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

		return &PriorityAction{
			Type:         ActionActivateAbility,
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

	// Phase 0A: If I have lethal, attack with exactly the lethal set.
	lethal := CalculateLethal(g, playerID)
	if lethal.IHaveLethal && len(lethal.LethalAttackers) > 0 {
		return lethal.LethalAttackers
	}

	// Phase 0B: Check race — if my clock < their clock, race aggressively.
	race := CalculateRace(g, playerID)

	var attackers []uuid.UUID
	for _, perm := range g.Battlefield {
		if perm.Controller != playerID {
			continue
		}
		if !perm.CanDeclareAsAttacker(g) {
			continue
		}
		if race.Racing && race.MyClock <= race.TheirClock {
			// Racing favorably: attack aggressively
			attackers = append(attackers, perm.ID())
		} else if shouldAttack(perm, g, opponentID, s.weights().Aggression) {
			attackers = append(attackers, perm.ID())
		}
	}
	return attackers
}

func (s *HeuristicStrategy) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	playerID := p.PlayerID()

	// Phase 0A: If opponent has lethal, block to survive even with bad trades.
	lethal := CalculateLethal(g, playerID)
	theyHaveLethal := lethal.TheyHaveLethal

	// Phase 0B: If racing favorably, only chump-block lethal.
	race := CalculateRace(g, playerID)
	racingFavorably := race.Racing && race.MyClock < race.TheirClock

	var assignments []mage.BlockAssignment

	var available []*mage.Permanent
	for _, perm := range g.Battlefield {
		if perm.Controller != playerID || !perm.CanDeclareAsBlocker(g) {
			continue
		}
		available = append(available, perm)
	}

	for _, group := range g.Combat.Groups {
		if group.DefenderID != playerID {
			continue
		}
		atk := g.FindPermanent(group.AttackerID)
		if atk == nil {
			continue
		}
		atkPow := atk.CurrentPower(g)

		// When facing lethal, block everything we can (ignore power threshold)
		if !theyHaveLethal {
			// When racing favorably, skip blocking (only block lethal)
			if racingFavorably {
				continue
			}
			if !shouldBlock(atkPow, g, p.PlayerID(), s.weights().BlockThreshold) {
				continue
			}
		}

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
			blkPow := blk.CurrentPower(g)
			atkTough := atk.CurrentToughness(g)

			// When facing lethal, block even with bad trades
			if theyHaveLethal {
				assignments = append(assignments, mage.BlockAssignment{
					BlockerID:  blk.ID(),
					AttackerID: atk.ID(),
				})
				available[i] = nil
				break
			}

			if blkPow >= atkTough || atkPow >= 3 {
				assignments = append(assignments, mage.BlockAssignment{
					BlockerID:  blk.ID(),
					AttackerID: atk.ID(),
				})
				available[i] = nil
				break
			}
		}
	}

	return assignments
}

func (s *HeuristicStrategy) autoSelectTargets(p mage.Player, g *mage.Game, card mage.Card) []uuid.UUID {
	playerID := p.PlayerID()

	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok {
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
					// Beneficial spell (buff, prevent damage, etc.): target own creatures first.
					var ownBest uuid.UUID
					ownBestScore := -1
					for _, id := range possible {
						perm := g.FindPermanent(id)
						if perm != nil && perm.Controller == playerID && perm.HasType(core.TypeCreature) {
							score := evalCreature(perm)
							if score > ownBestScore {
								ownBestScore = score
								ownBest = id
							}
						}
					}
					if ownBest != uuid.Nil {
						return []uuid.UUID{ownBest}
					}
					// Fall back to targeting self.
					for _, id := range possible {
						if id == playerID {
							return []uuid.UUID{id}
						}
					}
				}
				opponent := g.GetOpponent(playerID)
				if s.weights().TargetFace >= 0.5 && opponent != nil {
					// High TargetFace weight: prefer targeting the opponent player directly.
					for _, id := range possible {
						if id == opponent.PlayerID() {
							return []uuid.UUID{id}
						}
					}
				}
				// Compute spell damage for lethal-targeting decision.
				spellDamage := 0
				for _, ab := range card.Abilities() {
					if sa, ok := ab.(*mage.SpellAbility); ok {
						for _, e := range sa.Effects() {
							if dv := e.Properties().DamageValue; dv != nil {
								spellDamage = dv.Resolve(g, card.ID(), playerID)
							}
						}
					}
				}
				// Prefer lethal targets (highest ThreatPerMana), then fall through to raw threat.
				if spellDamage > 0 && opponent != nil {
					var bestLethalID uuid.UUID
					bestLethalTPM := -1.0
					for _, id := range possible {
						perm := g.FindPermanent(id)
						if perm != nil && perm.Controller == opponent.PlayerID() && perm.HasType(core.TypeCreature) {
							if spellDamage >= perm.CurrentToughness(g) {
								tpm := ThreatPerMana(perm)
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
				// Default: prefer the highest-threat opponent creature, then opponent player.
				var bestID uuid.UUID
				bestScore := -1
				for _, id := range possible {
					perm := g.FindPermanent(id)
					if perm != nil && perm.Controller != playerID && perm.HasType(core.TypeCreature) {
						score := evalCreature(perm)
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
					// Buff spell: target own most-threatening creature.
					var ownBest uuid.UUID
					ownBestScore := -1
					for _, id := range possible {
						perm := g.FindPermanent(id)
						if perm != nil && perm.Controller == playerID {
							score := evalCreature(perm)
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
				// Default/detriment: target highest-threat opponent creature.
				var bestID uuid.UUID
				bestScore := -1
				for _, id := range possible {
					perm := g.FindPermanent(id)
					if perm != nil && perm.Controller != playerID {
						score := evalCreature(perm)
						if score > bestScore {
							bestScore = score
							bestID = id
						}
					}
				}
				if bestID != uuid.Nil {
					return []uuid.UUID{bestID}
				}
				for _, id := range possible {
					perm := g.FindPermanent(id)
					if perm != nil && perm.Controller == playerID {
						return []uuid.UUID{id}
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

	return nil
}

// ─── AIPlayer ──────────────────────────────────────────────────────────────

// AIPlayer is a computer-controlled player with a pluggable AIStrategy.
type AIPlayer struct {
	*mage.BasePlayer
	strategy AIStrategy
}

// NewAIPlayer creates an AI player with the default Midrange personality.
func NewAIPlayer(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy:   NewHeuristicStrategy(MidrangeWeighted),
	}
}

// NewAggroAI creates an AI player with the Aggro personality.
func NewAggroAI(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy:   NewHeuristicStrategy(AggroWeighted),
	}
}

// NewControlAI creates an AI player with the Control personality.
func NewControlAI(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy:   NewHeuristicStrategy(ControlWeighted),
	}
}

// NewTempoAI creates an AI player with the Tempo personality.
func NewTempoAI(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy:   NewHeuristicStrategy(TempoWeighted),
	}
}

// NewWeightedAI creates an AI player with a custom WeightedPersonality.
func NewWeightedAI(name string, w WeightedPersonality) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy:   NewHeuristicStrategy(w),
	}
}

// ChooseMode implements the Player interface for AI mode selection.
func (ai *AIPlayer) ChooseMode(modes []string, reason string) int {
	switch reason {
	case "Healing Salve":
		if ai.Life() <= 10 {
			return 0
		}
		return 1
	}
	return 0
}

// GetPriorityAction decides what the AI should do when it has priority.
func (ai *AIPlayer) GetPriorityAction(g *mage.Game, landsPlayed int, mainPhase bool) PriorityAction {
	return ai.strategy.PriorityAction(ai.BasePlayer, g, landsPlayed, mainPhase)
}

// AIAttackers returns the IDs of creatures the AI wants to attack with.
func (ai *AIPlayer) AIAttackers(g *mage.Game) []uuid.UUID {
	return ai.strategy.Attackers(ai.BasePlayer, g)
}

// AIBlockers returns a blocker->attacker mapping for the AI's blocking decisions.
func (ai *AIPlayer) AIBlockers(g *mage.Game) []mage.BlockAssignment {
	return ai.strategy.Blockers(ai.BasePlayer, g)
}

// DeclareAttackers implements the Player interface for AI.
func (ai *AIPlayer) DeclareAttackers(g *mage.Game) []uuid.UUID {
	return ai.AIAttackers(g)
}

// DeclareBlockers implements the Player interface for AI.
func (ai *AIPlayer) DeclareBlockers(g *mage.Game) []mage.BlockAssignment {
	return ai.AIBlockers(g)
}

// NewBurnAI creates an AI player focused on dealing direct damage to the opponent's face.
func NewBurnAI(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy:   NewHeuristicStrategy(BurnWeighted),
	}
}

// NewAdaptiveAI creates an AI that plays aggressively when ahead and switches
// to a controlling game plan when behind on life.
func NewAdaptiveAI(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy: &AdaptiveStrategy{
			Aggressive: NewHeuristicStrategy(AggroWeighted),
			Defensive:  NewHeuristicStrategy(ControlWeighted),
		},
	}
}

// ─── Helper functions ───────────────────────────────────────────────────────

// shouldAttack decides whether to attack with a creature based on the
// continuous Aggression weight (0.0 to 1.0).
//   - At 1.0: attack with everything (ignore profitability).
//   - At 0.0: only attack when guaranteed profitable.
//   - At intermediate values: attack if profitable, or if aggression exceeds
//     the risk threshold from the trade analysis.
func shouldAttack(atk *mage.Permanent, g *mage.Game, opponentID uuid.UUID, aggression float64) bool {
	// At maximum aggression, always attack.
	if aggression >= 1.0 {
		return true
	}
	// Check base profitability (attacker survives or trades up).
	if profitableToAttack(atk, g, opponentID) {
		return true
	}
	// At aggression >= 0.7, also attack on equal trades (same CMC).
	if aggression >= 0.7 {
		return marginallyProfitableToAttack(atk, g, opponentID)
	}
	return false
}

// marginallyProfitableToAttack returns true if the attacker would trade evenly
// (same CMC) or if no blocker exists. This is less strict than profitableToAttack.
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
	// At least trades (kills the blocker)
	return atkPow >= blkTough
}

// shouldBlock decides whether to block an attacker with the given power,
// based on the continuous BlockThreshold weight (0.0 to 1.0).
//   - At 0.0: block everything (minimum power threshold = 0).
//   - At 0.3: block attackers with power >= 3.
//   - At 0.7: block attackers with power >= 7.
//   - At >= 0.95: block almost nothing (effectively never block).
//
// The mapping is: minPowerToBlock = BlockThreshold * 10, so the weight
// maps directly to an absolute power threshold.
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

// profitableToAttack returns true if attacking with atk is unlikely to result
// in an unfavorable trade. It finds the best blocker the opponent could assign
// and checks whether the attacker survives or at least kills something of equal
// or greater mana value.
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

	// No blocker available — attack is free.
	if bestBlocker == nil {
		return true
	}

	atkPow := atk.CurrentPower(g)
	atkTough := atk.CurrentToughness(g)
	blkPow := bestBlocker.CurrentPower(g)
	blkTough := bestBlocker.CurrentToughness(g)

	atkSurvives := blkPow < atkTough
	blkDies := atkPow >= blkTough

	// Attacker survives the block → always attack.
	if atkSurvives {
		return true
	}
	// Both die (trade) → only trade if the blocker costs at least as much mana.
	if blkDies {
		return bestBlocker.Card.ManaCost().CMC() >= atk.Card.ManaCost().CMC()
	}
	// Attacker dies, blocker survives → don't attack.
	return false
}

// ─── AdaptiveStrategy ───────────────────────────────────────────────────────

// AdaptiveStrategy switches between two strategies based on relative life totals.
// When the AI is ahead on life it plays aggressively; when behind it plays defensively.
type AdaptiveStrategy struct {
	Aggressive AIStrategy
	Defensive  AIStrategy
}

func (s *AdaptiveStrategy) active(p mage.Player, g *mage.Game) AIStrategy {
	if DefaultEvaluator(g, p.PlayerID()) >= 0 {
		return s.Aggressive
	}
	return s.Defensive
}

func (s *AdaptiveStrategy) PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) PriorityAction {
	return s.active(p, g).PriorityAction(p, g, landsPlayed, mainPhase)
}

func (s *AdaptiveStrategy) Attackers(p mage.Player, g *mage.Game) []uuid.UUID {
	return s.active(p, g).Attackers(p, g)
}

func (s *AdaptiveStrategy) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	return s.active(p, g).Blockers(p, g)
}

// ─── SequentialStrategy ─────────────────────────────────────────────────────

// SequentialStrategy tries each sub-strategy in order and uses the first one
// that produces a non-pass priority action. Attackers and blockers are taken
// from the first strategy that returns non-empty results.
type SequentialStrategy struct {
	Strategies []AIStrategy
}

func (s *SequentialStrategy) PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) PriorityAction {
	for _, strat := range s.Strategies {
		if action := strat.PriorityAction(p, g, landsPlayed, mainPhase); action.Type != ActionPass {
			return action
		}
	}
	return PriorityAction{Type: ActionPass}
}

func (s *SequentialStrategy) Attackers(p mage.Player, g *mage.Game) []uuid.UUID {
	for _, strat := range s.Strategies {
		if atks := strat.Attackers(p, g); len(atks) > 0 {
			return atks
		}
	}
	return nil
}

func (s *SequentialStrategy) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	for _, strat := range s.Strategies {
		if blks := strat.Blockers(p, g); len(blks) > 0 {
			return blks
		}
	}
	return nil
}
