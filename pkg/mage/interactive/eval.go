package interactive

import (
	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// BoardScore returns a heuristic board evaluation for playerID. Positive values
// mean the player is ahead; negative values mean they are behind.
//
// Components:
//   - Permanent score: +threatScore for own permanents, -threatScore for opponents
//   - Life advantage: (myLife - opponentLife) / 4
//   - Hand advantage: (len(myHand) - len(opponentHand)) * 2
func BoardScore(playerID uuid.UUID, g *mage.Game) int {
	opponent := g.GetOpponent(playerID)
	if opponent == nil {
		return 0
	}
	opponentID := opponent.PlayerID()

	score := 0
	for _, perm := range g.Battlefield {
		if perm.Controller == playerID {
			score += threatScore(perm, g)
		} else if perm.Controller == opponentID {
			score -= threatScore(perm, g)
		}
	}

	myPlayer := g.GetPlayer(playerID)
	if myPlayer != nil {
		score += (myPlayer.Life() - opponent.Life()) / 4
		score += (len(myPlayer.Hand()) - len(opponent.Hand())) * 2
	}
	return score
}

// ThreatPerMana returns a permanent's threat-per-mana-spent ratio.
// Returns 0 if the permanent has CMC 0 (tokens, lands).
// Used by autoSelectTargets to prefer the most mana-efficient threat.
func ThreatPerMana(perm *mage.Permanent, g *mage.Game) float64 {
	cmc := perm.Card.ManaCost().CMC()
	if cmc == 0 {
		return 0
	}
	return float64(threatScore(perm, g)) / float64(cmc)
}

// spellIsWorthless returns true if the spell requires at least one target but
// has no valid targets available right now. Used to skip casting a targeted
// sorcery when there is nothing to hit (avoids a main-phase cast bug where
// the AI commits to casting but then has nil targets).
func spellIsWorthless(card mage.Card, p mage.Player, g *mage.Game) bool {
	playerID := p.PlayerID()
	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok {
			continue
		}
		for _, t := range sa.Targets() {
			if len(t.Possible(playerID, card, g)) == 0 {
				return true
			}
		}
	}
	return false
}

// spellValue computes a context-sensitive score for a spell in the AI's hand,
// used to pick the best spell to cast each turn.
//
// Scoring rules (applied to each effect in the spell's SpellAbility):
//   - Creature:          basePower*2 + baseToughness (fallback: CMC)
//   - DrawCount > 0:     +DrawCount*3
//   - Damage > 0, detriment: +Damage, +2 per opponent creature it would kill
//   - Mass == true:      +20 if behind on board (BoardScore < 0), -10 if ahead
//   - Detriment, targeted, no damage: +threatScore of best reachable opponent permanent
//   - Fallback:          CMC
func spellValue(card mage.Card, p mage.Player, g *mage.Game) int {
	playerID := p.PlayerID()
	cmc := card.ManaCost().CMC()

	// Creatures: P/T heuristic
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

	for _, a := range card.Abilities() {
		sa, ok := a.(*mage.SpellAbility)
		if !ok {
			continue
		}
		for _, e := range sa.Effects() {
			props := e.Properties()

			if props.DrawCount > 0 {
				score += props.DrawCount * 3
				found = true
			}

			if props.Damage > 0 && props.Outcome == mage.OutcomeDetriment {
				score += props.Damage
				// Bonus for lethal hits against opponent creatures
				opponent := g.GetOpponent(playerID)
				if opponent != nil {
					for _, perm := range g.Battlefield {
						if perm.Controller == opponent.PlayerID() && perm.HasType(core.TypeCreature) {
							if props.Damage >= perm.CurrentToughness(g) {
								score += 2
								break // count the lethality bonus once
							}
						}
					}
				}
				found = true
			}

			if props.Mass {
				if BoardScore(playerID, g) < 0 {
					score += 20
				} else {
					score -= 10
				}
				found = true
			}

			// Pure targeted removal (destroy, exile, bounce, etc.): value by best target
			if props.Outcome == mage.OutcomeDetriment && !props.Mass && props.Damage == 0 {
				opponent := g.GetOpponent(playerID)
				if opponent != nil {
					bestTS := 0
					for _, perm := range g.Battlefield {
						if perm.Controller == opponent.PlayerID() {
							ts := threatScore(perm, g)
							if ts > bestTS {
								bestTS = ts
							}
						}
					}
					if bestTS > 0 {
						score += bestTS
						found = true
					}
				}
			}
		}
	}

	if found && score > 0 {
		return score
	}
	return cmc
}
