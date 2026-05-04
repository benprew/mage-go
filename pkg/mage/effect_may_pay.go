package mage

import (
	"fmt"

	"github.com/google/uuid"
)

// mayPayManaEffect implements "you may pay {N}. If you do, ___" mid-resolution
// (CR 603.4-style optional cost during a triggered ability's resolution).
//
// On Apply, the controller is asked whether to pay the mana cost
// (Player.ChooseMayAbility on a description like "pay {1}{W} to put X +1/+1
// counters on target Unicorn"). If the player accepts AND the mana can be
// paid via Game.TryPayCostFromLands, the inner Effect resolves with the same
// sourceID/controller/targets the outer trigger resolved with. Otherwise
// nothing happens.
//
// Used by Cradle of Vitality, Kels Fight Fixer, Emiel the Blessed,
// Drainpipe Vermin, and similar "may pay" triggered abilities.
type mayPayManaEffect struct {
	cost        string
	description string
	inner       Effect
}

// MayPayMana wraps an inner Effect with an optional mana payment. If the
// controller chooses to pay and can pay the cost (using TryPayCostFromLands),
// the inner effect resolves with the same parameters the outer effect was
// called with. If the controller declines or cannot pay, the inner effect
// does not resolve.
//
// The description is shown to the player (and used by AI search heuristics)
// to decide whether to accept the payment; it should briefly describe what
// they get if they pay.
func MayPayMana(cost, description string, inner Effect) Effect {
	return &mayPayManaEffect{
		cost:        cost,
		description: description,
		inner:       inner,
	}
}

func (e *mayPayManaEffect) Text() string {
	return fmt.Sprintf("you may pay %s. If you do, %s", e.cost, e.inner.Text())
}

func (e *mayPayManaEffect) Properties() EffectProperties {
	// Defer to the inner effect's outcome; the may-pay wrapper itself is
	// neither beneficial nor detrimental until the inner runs.
	return e.inner.Properties()
}

func (e *mayPayManaEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	p := g.GetPlayer(controller)
	if p == nil {
		return nil
	}
	if !p.ChooseMayAbility(e.description) {
		return nil
	}
	if !g.TryPayCostFromLands(controller, e.cost) {
		return nil
	}
	return e.inner.Apply(g, sourceID, controller, targets)
}
