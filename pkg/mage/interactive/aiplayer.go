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
type Personality struct {
	Name                string
	CastOrder           SpellOrder
	HoldInstants        bool // if true, only cast instants in response to opponent actions
	AttackAll           bool // if true, attack with every eligible creature; if false, only attack when profitable
	BlockPowerThreshold int  // block attackers with power >= this (0 = block everything)
	TargetFace          bool // if true, damage spells target the opponent player rather than creatures
}

// Preset personalities.
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
type HeuristicStrategy struct {
	Personality Personality
}

func (s *HeuristicStrategy) PriorityAction(p mage.Player, g *mage.Game, landsPlayed int, mainPhase bool) PriorityAction {
	playerID := p.PlayerID()

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

		var bestCard mage.Card
		bestScore := -1
		for _, card := range g.GetCastableSpells(playerID) {
			if card.HasType(core.TypeInstant) {
				continue
			}
			score := spellValue(card, p, g)
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

	// HoldInstants: only skip instants during main phase; cast freely during
	// opponent's turn / responses to the stack.
	if !s.Personality.HoldInstants || !mainPhase {
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

func (s *HeuristicStrategy) Attackers(p mage.Player, g *mage.Game) []uuid.UUID {
	opponent := g.GetOpponent(p.PlayerID())
	var opponentID uuid.UUID
	if opponent != nil {
		opponentID = opponent.PlayerID()
	}

	var attackers []uuid.UUID
	for _, perm := range g.Battlefield {
		if perm.Controller != p.PlayerID() {
			continue
		}
		if !perm.CanDeclareAsAttacker(g) {
			continue
		}
		if s.Personality.AttackAll || profitableToAttack(perm, g, opponentID) {
			attackers = append(attackers, perm.ID())
		}
	}
	return attackers
}

func (s *HeuristicStrategy) Blockers(p mage.Player, g *mage.Game) []mage.BlockAssignment {
	var assignments []mage.BlockAssignment

	var available []*mage.Permanent
	for _, perm := range g.Battlefield {
		if perm.Controller != p.PlayerID() || !perm.CanDeclareAsBlocker(g) {
			continue
		}
		available = append(available, perm)
	}

	for _, group := range g.Combat.Groups {
		if group.DefenderID != p.PlayerID() {
			continue
		}
		atk := g.FindPermanent(group.AttackerID)
		if atk == nil {
			continue
		}
		atkPow := atk.CurrentPower(g)
		if s.Personality.BlockPowerThreshold > 0 && atkPow < s.Personality.BlockPowerThreshold {
			continue
		}

		for i, blk := range available {
			if blk == nil {
				continue
			}
			if !mage.CanBlock(blk, atk, g) {
				continue
			}
			if mage.HasLandwalkEvasion(atk, p.PlayerID(), g) {
				continue
			}
			blkPow := blk.CurrentPower(g)
			atkTough := atk.CurrentToughness(g)

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
				if s.Personality.TargetFace && opponent != nil {
					// Burn strategy: always target the opponent player directly.
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
		strategy:   &HeuristicStrategy{Personality: MidrangePersonality},
	}
}

// NewAggroAI creates an AI player with the Aggro personality.
func NewAggroAI(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy:   &HeuristicStrategy{Personality: AggroPersonality},
	}
}

// NewControlAI creates an AI player with the Control personality.
func NewControlAI(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy:   &HeuristicStrategy{Personality: ControlPersonality},
	}
}

// NewTempoAI creates an AI player with the Tempo personality.
func NewTempoAI(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy:   &HeuristicStrategy{Personality: TempoPersonality},
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
		strategy:   &HeuristicStrategy{Personality: BurnPersonality},
	}
}

// NewAdaptiveAI creates an AI that plays aggressively when ahead and switches
// to a controlling game plan when behind on life.
func NewAdaptiveAI(name string) *AIPlayer {
	return &AIPlayer{
		BasePlayer: mage.NewBasePlayer(name),
		strategy: &AdaptiveStrategy{
			Aggressive: &HeuristicStrategy{Personality: AggroPersonality},
			Defensive:  &HeuristicStrategy{Personality: ControlPersonality},
		},
	}
}

// ─── Helper functions ───────────────────────────────────────────────────────

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
