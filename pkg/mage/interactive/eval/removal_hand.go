package eval

import (
	"cmp"
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
)

// RemovalHandValue adds the value of future answers to the normal hand score.
func RemovalHandValue(g *mage.Game, p mage.Player) float64 {
	values := make([]float64, 0, len(p.Hand()))
	for _, card := range p.Hand() {
		if value := RemovalCardValue(g, p, card); value > 0 {
			values = append(values, value)
		}
	}
	slices.SortFunc(values, func(a, b float64) int { return cmp.Compare(b, a) })
	value := 0.0
	for i, capacity := range values {
		// Additional answers reduce the cost of spending one answer.
		value += capacity / float64(i+1)
	}
	return value
}

// RemovalCardValue scores an answer's future use from its effects and targets.
func RemovalCardValue(g *mage.Game, p mage.Player, card *mage.Card) float64 {
	best := 0.0
	mana := developedMana(g, p)
	for _, ability := range card.AllAbilities() {
		action, ok := ability.(*mage.ActionDefinition)
		if !ok || action.Kind() != mage.ActionSpell {
			continue
		}
		flexibility := 0.0
		damageTarget := false
		for _, target := range action.Targets() {
			switch target := target.(type) {
			case *mage.CreatureTarget:
				capacity := 4.0
				if len(target.Filters) > 0 {
					capacity = 3
				}
				flexibility = max(flexibility, capacity)
			case *mage.PermanentTarget:
				if len(target.Filters) == 0 {
					flexibility = max(flexibility, 5)
				}
			case *mage.DamageAnyTarget:
				flexibility = max(flexibility, 4)
				damageTarget = true
			}
		}
		if flexibility == 0 {
			continue
		}
		purpose := TargetPurposeForEffects(action.Effects())
		for _, effect := range action.Effects() {
			props := effect.Properties()
			if props.Mass {
				continue
			}
			if props.DamageValue != nil && props.Outcome == mage.OutcomeDetriment {
				ctx := mage.NewEffectContext(g, card.ID(), p.PlayerID(), nil)
				if card.ManaCost().HasX {
					ctx.XValue = max(mana-card.ManaCost().CMC(), 0)
				}
				best = max(best, float64(props.DamageValue.Resolve(ctx)))
				continue
			}
			if damageTarget {
				if card.ManaCost().HasX && props.Outcome == mage.OutcomeDetriment {
					best = max(best, float64(max(mana-card.ManaCost().CMC(), 0)))
				}
				continue
			}
			if props.ToughnessBoost < 0 {
				best = max(best, float64(-props.ToughnessBoost))
				continue
			}
			if props.PowerBoost < 0 {
				continue
			}
			if purpose == TargetRemoval || purpose == TargetExile {
				value := flexibility
				// A gradual decline avoids a sudden change in casting decisions.
				early := min(1, max(0, float64(8-g.CurrentTurn())/4))
				value *= 1 + early
				best = max(best, value)
			}
		}
	}
	// A costly answer has less value while its mana sources are missing.
	return best * min(1, float64(mana+1)/float64(max(card.ManaCost().CMC(), 1)))
}

func developedMana(g *mage.Game, p mage.Player) int {
	mana := 0
	for _, permanent := range g.AllBattlefield() {
		if permanent.ControllerID() != p.PlayerID() {
			continue
		}
		amount := 0
		for _, ability := range permanent.RuntimeAbilities {
			produced := 0
			for _, production := range mage.ManaProductionsForAbilityInGame(ability, g, permanent.ID()) {
				produced += max(production.Amount, 1)
			}
			amount = max(amount, produced)
		}
		mana += amount
	}
	return mana
}
