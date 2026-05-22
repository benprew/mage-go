package eval

import (
	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// ThreatPerMana returns a permanent's threat-per-mana-spent ratio.
// Uses base stats only. Prefer ThreatPerManaInGame when a *Game is available.
func ThreatPerMana(perm *mage.Permanent) float64 {
	cmc := perm.Card.ManaCost().CMC()
	if cmc == 0 {
		return 0
	}
	return float64(EvalCreature(perm)) / float64(cmc)
}

// ThreatPerManaInGame returns a permanent's threat-per-mana-spent ratio,
// using CurrentPower/CurrentToughness for accurate continuous-effect-aware scoring.
func ThreatPerManaInGame(perm *mage.Permanent, g *mage.Game) float64 {
	cmc := perm.Card.ManaCost().CMC()
	if cmc == 0 {
		return 0
	}
	return float64(EvalCreatureInGame(perm, g)) / float64(cmc)
}

// SpellIsWorthless returns true if the spell requires at least one target but
// has no valid targets available right now.
func SpellIsWorthless(card mage.Card, p mage.Player, g *mage.Game) bool {
	playerID := p.PlayerID()
	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok || sa.Kind() != mage.ActionSpell {
			continue
		}
		for _, t := range sa.Targets() {
			if len(t.Possible(playerID, card, g)) == 0 {
				return true
			}
		}
	}
	for _, t := range card.CastTargets() {
		if len(t.Possible(playerID, card, g)) == 0 {
			return true
		}
	}
	return false
}

// SpellValue computes a context-sensitive score for a spell in the AI's hand.
// Board-relative: mass removal is worth more when behind, damage spells get
// lethal-enabling bonuses, and draw spells scale with hand emptiness.
func SpellValue(card mage.Card, p mage.Player, g *mage.Game) int {
	playerID := p.PlayerID()
	cmc := card.ManaCost().CMC()

	if card.HasType(core.TypeCreature) {
		pw := card.Power()
		tg := card.Toughness()
		v := pw*2 + tg
		if v > 0 {
			return v
		}
		return cmc
	}

	score := 0
	found := false

	opponent := g.GetOpponent(playerID)
	var oppID uuid.UUID
	if opponent != nil {
		oppID = opponent.PlayerID()
	}

	// Count creatures per side for board-relative scoring.
	myCreatures, oppCreatures := 0, 0
	var oppBestScore int
	for _, perm := range g.AllBattlefield() {
		if !perm.HasType(core.TypeCreature) {
			continue
		}
		switch perm.Controller {
		case playerID:
			myCreatures++
		case oppID:
			oppCreatures++
			ts := EvalCreatureInGame(perm, g)
			if ts > oppBestScore {
				oppBestScore = ts
			}
		}
	}

	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok || sa.Kind() != mage.ActionSpell {
			continue
		}
		for _, hint := range sa.AIHints() {
			if hintScore := aiActionHintScore(hint, card, p, g); hintScore != 0 {
				score += hintScore
				found = true
			}
		}
		for _, e := range sa.Effects() {
			props := e.Properties()
			if hintScore := aiHintScore(props, card, p, g); hintScore != 0 {
				score += hintScore
				found = true
			}

			if props.DrawCount > 0 {
				drawScore := props.DrawCount * 3
				// Draw is more valuable when hand is empty.
				handSize := len(p.Hand())
				if handSize <= 2 {
					drawScore += props.DrawCount * 2
				} else if handSize <= 4 {
					drawScore += props.DrawCount
				}
				score += drawScore
				found = true
			}

			if props.DamageValue != nil && props.Outcome == mage.OutcomeDetriment {
				dmg := props.DamageValue.Resolve(g, card.ID(), playerID, nil)
				score += dmg
				if opponent != nil {
					// Bonus for being able to kill an opponent creature.
					bestLethalBonus := 0
					for _, perm := range g.AllBattlefield() {
						if perm.Controller == oppID && perm.HasType(core.TypeCreature) {
							if dmg >= perm.CurrentToughness(g) {
								bonus := max(EvalCreatureInGame(perm, g)/2, 2)
								if bonus > bestLethalBonus {
									bestLethalBonus = bonus
								}
							}
						}
					}
					score += bestLethalBonus

					// Bonus if this spell enables lethal next attack.
					if dmg >= opponent.Life() {
						score += 15
					} else {
						// Check if burn to face + existing attackers = lethal.
						pushThrough, _ := estimatePushThroughDamage(g, playerID, oppID)
						if pushThrough+dmg >= opponent.Life() {
							score += 10
						}
					}
				}
				found = true
			}

			if props.Mass && props.Outcome == mage.OutcomeDetriment {
				// Board-relative: scale mass removal with board disadvantage.
				creatureDiff := oppCreatures - myCreatures
				if creatureDiff >= 3 {
					score += 25
				} else if creatureDiff >= 1 {
					score += 15 + creatureDiff*3
				} else if creatureDiff == 0 {
					score += 5
				} else {
					// We have more creatures — wrath hurts us.
					score -= 5 + (-creatureDiff)*3
				}
				found = true
			}

			if props.Outcome == mage.OutcomeDetriment && !props.Mass && props.DamageValue == nil {
				if oppBestScore > 0 {
					score += oppBestScore
					found = true
				}
			}

			if props.Outcome == mage.OutcomeBenefit && !props.Mass {
				if props.LifeGain > 0 {
					me := g.GetPlayer(playerID)
					if me != nil && me.Life() <= 10 {
						score += props.LifeGain
					} else {
						score += props.LifeGain / 2
					}
					found = true
				}
				if props.PowerBoost > 0 || props.ToughnessBoost > 0 {
					score += props.PowerBoost*2 + props.ToughnessBoost
					found = true
				}
				if props.TokenPower > 0 || props.TokenToughness > 0 {
					score += props.TokenPower*2 + props.TokenToughness
					found = true
				}
			}

			if props.IsBounce {
				if oppBestScore > 0 {
					// Bounce is worth slightly less than destroy (they can replay).
					score += oppBestScore * 3 / 4
					found = true
				}
			}
		}
	}

	if found {
		return score
	}
	return cmc
}

func aiHintScore(props mage.EffectProperties, card mage.Card, p mage.Player, g *mage.Game) int {
	score := props.ValueBias
	for _, role := range props.AIRoles {
		score += aiRoleScore(role, props, card, p, g)
	}
	return score
}

func aiActionHintScore(hint mage.AIHint, card mage.Card, p mage.Player, g *mage.Game) int {
	props := mage.EffectProperties{AIRoles: hint.Roles, ValueBias: hint.ValueBias}
	return aiHintScore(props, card, p, g)
}

func aiRoleScore(role mage.AIRole, props mage.EffectProperties, card mage.Card, p mage.Player, g *mage.Game) int {
	switch role {
	case mage.AIRoleRemoval:
		return 7
	case mage.AIRoleBurn:
		if props.DamageValue != nil {
			return props.DamageValue.Resolve(g, card.ID(), p.PlayerID(), nil)
		}
		return 5
	case mage.AIRolePump, mage.AIRoleCombatTrick:
		return 4
	case mage.AIRoleProtection:
		return 3
	case mage.AIRoleCardDraw:
		handSize := len(p.Hand())
		if handSize <= 2 {
			return 8
		}
		if handSize <= 4 {
			return 6
		}
		return 4
	case mage.AIRoleManaSink:
		return max(2, evalAvailableMana(g, p.PlayerID())-card.ManaCost().CMC())
	case mage.AIRoleFinisher:
		return 8
	case mage.AIRoleEngine:
		return 6
	default:
		return 0
	}
}

func evalAvailableMana(g mage.GameReader, playerID uuid.UUID) int {
	return CountAvailableMana(g, playerID)
}

// CountAvailableMana counts the total mana available from untapped sources a player controls.
func CountAvailableMana(g mage.GameReader, playerID uuid.UUID) int {
	return g.HypotheticalMana(playerID)
}

// HandCMCs returns the CMC of each non-land, non-instant card in a player's hand.
func HandCMCs(hand []mage.Card) []int {
	var cmcs []int
	for _, c := range hand {
		if c.HasType(core.TypeLand) || c.HasType(core.TypeInstant) {
			continue
		}
		cmcs = append(cmcs, c.ManaCost().CMC())
	}
	return cmcs
}

// ManaCurveBonus scores a spell based on how well it uses available mana.
func ManaCurveBonus(cardCMC, availableMana int, handCMCs []int) float64 {
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
