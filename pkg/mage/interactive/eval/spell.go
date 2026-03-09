package eval

import (
	"github.com/google/uuid"
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// ThreatPerMana returns a permanent's threat-per-mana-spent ratio.
func ThreatPerMana(perm *mage.Permanent) float64 {
	cmc := perm.Card.ManaCost().CMC()
	if cmc == 0 {
		return 0
	}
	return float64(EvalCreature(perm)) / float64(cmc)
}

// SpellIsWorthless returns true if the spell requires at least one target but
// has no valid targets available right now.
func SpellIsWorthless(card mage.Card, p mage.Player, g *mage.Game) bool {
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

// SpellValue computes a context-sensitive score for a spell in the AI's hand.
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

			if props.DamageValue != nil && props.Outcome == mage.OutcomeDetriment {
				dmg := props.DamageValue.Resolve(g, card.ID(), playerID)
				score += dmg
				opponent := g.GetOpponent(playerID)
				if opponent != nil {
					for _, perm := range g.Battlefield {
						if perm.Controller == opponent.PlayerID() && perm.HasType(core.TypeCreature) {
							if dmg >= perm.CurrentToughness(g) {
								score += 2
								break
							}
						}
					}
				}
				found = true
			}

			if props.Mass {
				if DefaultEvaluator(g, playerID) < 0 {
					score += 20
				} else {
					score -= 10
				}
				found = true
			}

			if props.Outcome == mage.OutcomeDetriment && !props.Mass && props.DamageValue == nil {
				opponent := g.GetOpponent(playerID)
				if opponent != nil {
					bestTS := 0
					for _, perm := range g.Battlefield {
						if perm.Controller == opponent.PlayerID() {
							ts := EvalCreature(perm)
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

// CountAvailableMana counts the number of untapped mana sources a player controls.
func CountAvailableMana(g *mage.Game, playerID uuid.UUID) int {
	count := 0
	for _, perm := range g.Battlefield {
		if perm.Controller != playerID {
			continue
		}
		if perm.Tapped {
			continue
		}
		if perm.HasType(core.TypeLand) {
			count++
			continue
		}
		for _, a := range perm.RuntimeAbilities {
			if _, ok := mage.UnwrapAbility(a).(*mage.ManaAbility); ok {
				count++
				break
			}
		}
	}
	return count
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
