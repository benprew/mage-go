package ai

import (
	"strings"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

// ChooseModeHeuristic selects a mode for a modal spell based on the mode
// descriptions and the player's current game state (life total, hand size,
// board state).
func ChooseModeHeuristic(modes []string, life, handSize int) int {
	bestMode := 0
	bestScore := -1

	for i, mode := range modes {
		lower := strings.ToLower(mode)
		score := scoreModeText(lower, life, handSize)

		if score > bestScore {
			bestScore = score
			bestMode = i
		}
	}

	return bestMode
}

// ChooseModeWithContext selects a mode using full game state for board-relative
// evaluation. Falls back to text heuristic when game context is unavailable.
func ChooseModeWithContext(modes []string, g *mage.Game, playerID uuid.UUID) int {
	me := g.GetPlayer(playerID)
	if me == nil {
		return ChooseModeHeuristic(modes, 20, 3)
	}

	opponent := g.GetOpponent(playerID)
	life := me.Life()
	handSize := len(me.Hand())

	// Count board state for context.
	myCreatures, oppCreatures := 0, 0
	for _, perm := range g.AllBattlefield() {
		if !perm.HasType(core.TypeCreature) {
			continue
		}
		if perm.Controller == playerID {
			myCreatures++
		} else if opponent != nil && perm.Controller == opponent.PlayerID() {
			oppCreatures++
		}
	}

	bestMode := 0
	bestScore := -1

	for i, mode := range modes {
		lower := strings.ToLower(mode)
		score := scoreModeText(lower, life, handSize)

		// Board-relative adjustments.
		if strings.Contains(lower, "destroy") || strings.Contains(lower, "exile") {
			if oppCreatures > myCreatures {
				score += 3
			}
			if oppCreatures == 0 {
				score -= 4 // no targets
			}
		}

		if strings.Contains(lower, "token") || strings.Contains(lower, "creature") {
			if strings.Contains(lower, "create") || strings.Contains(lower, "put") {
				if myCreatures < oppCreatures {
					score += 2
				}
			}
		}

		if strings.Contains(lower, "damage") || strings.Contains(lower, "deal") {
			if opponent != nil && opponent.Life() <= 5 {
				score += 3 // closer to lethal
			}
		}

		if score > bestScore {
			bestScore = score
			bestMode = i
		}
	}

	return bestMode
}

// ChooseModeWithEffects selects a mode using effect metadata first and mode
// labels as a fallback.
func ChooseModeWithEffects(modes []mage.Mode, g *mage.Game, playerID uuid.UUID) int {
	labels := make([]string, len(modes))
	for i, mode := range modes {
		labels[i] = mode.Label
	}
	if len(modes) == 0 || g == nil {
		return ChooseModeHeuristic(labels, 20, 3)
	}

	me := g.GetPlayer(playerID)
	life, handSize := 20, 3
	if me != nil {
		life = me.Life()
		handSize = len(me.Hand())
	}

	bestMode := 0
	bestScore := -1
	for i, mode := range modes {
		score := scoreModeText(strings.ToLower(mode.Label), life, handSize)
		for _, effect := range mode.Effects {
			score += scoreModeEffectProperties(effect.Properties(), life, handSize)
		}
		if score > bestScore {
			bestScore = score
			bestMode = i
		}
	}
	return bestMode
}

func scoreModeEffectProperties(props mage.EffectProperties, life, handSize int) int {
	score := props.ValueBias
	for _, role := range props.AIRoles {
		switch role {
		case mage.AIRoleRemoval, mage.AIRoleBurn:
			score += 7
		case mage.AIRoleCardDraw:
			if handSize <= 2 {
				score += 9
			} else {
				score += 6
			}
		case mage.AIRolePump, mage.AIRoleCombatTrick:
			score += 4
		case mage.AIRoleProtection:
			score += 3
		case mage.AIRoleFinisher, mage.AIRoleEngine:
			score += 6
		}
	}
	if props.DrawCount > 0 {
		score += props.DrawCount * 3
		if handSize <= 2 {
			score += props.DrawCount * 2
		}
	}
	if props.DamageValue != nil && props.Outcome == mage.OutcomeDetriment {
		score += 7
	}
	if props.Outcome == mage.OutcomeDetriment && props.DamageValue == nil {
		score += 6
	}
	if props.LifeGain > 0 {
		if life <= 10 {
			score += props.LifeGain
		} else {
			score += props.LifeGain / 2
		}
	}
	if props.TokenPower > 0 || props.TokenToughness > 0 {
		score += props.TokenPower*2 + props.TokenToughness
	}
	return score
}

// scoreModeText scores a mode based on text content and basic game state.
func scoreModeText(lower string, life, handSize int) int {
	var score int

	switch {
	case strings.Contains(lower, "destroy") || strings.Contains(lower, "exile"):
		score = 8
	case strings.Contains(lower, "damage") || strings.Contains(lower, "deal"):
		score = 7
	case strings.Contains(lower, "draw"):
		score = 6
		if handSize < 3 {
			score = 9
		}
	case strings.Contains(lower, "return") && strings.Contains(lower, "hand"):
		score = 5 // bounce
	case strings.Contains(lower, "counter"):
		score = 7
	case strings.Contains(lower, "gain") && strings.Contains(lower, "life"):
		score = 4
		if life <= 10 {
			score = 9
		}
	case strings.Contains(lower, "prevent"):
		score = 3
		if life <= 10 {
			score = 5
		}
	case strings.Contains(lower, "token") || strings.Contains(lower, "create"):
		score = 5
	case strings.Contains(lower, "+") || strings.Contains(lower, "gets"):
		score = 4
	default:
		score = 1
	}

	return score
}

// ChooseModeForAI picks a mode using game context when available.
func ChooseModeForAI(modes []string, g *mage.Game, playerID uuid.UUID) int {
	if g != nil {
		return ChooseModeWithContext(modes, g, playerID)
	}
	return ChooseModeHeuristic(modes, 20, 3)
}

// Ensure packages are used.
var (
	_ = eval.EvalCreatureInGame
	_ uuid.UUID
)
