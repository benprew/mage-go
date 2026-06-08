package mage

import (
	"fmt"
)

// sacrificeUnlessPayManaEffect implements an upkeep-style "do X unless you pay
// {cost}" branch where the payment is OPTIONAL (CR 118.3 / 603.4). The
// controller is asked whether to pay; only if they accept AND the mana can be
// paid does the source survive. If they decline or cannot pay, the source is
// sacrificed.
//
// This differs from a forced TryPayCostFromLands branch: the player keeps the
// choice of letting the permanent go even when they could afford the cost
// (e.g. Junun Efreet, Lake of the Dead's upkeep on Eternal Flame, etc.).
type sacrificeUnlessPayManaEffect struct {
	cost   string
	prompt string
}

// SacrificeUnlessPayMana wraps an upkeep trigger so the controller may pay the
// given mana cost to keep the source. Declining (or being unable to pay)
// sacrifices the source. The prompt is shown via Player.ChooseMayAbility.
func SacrificeUnlessPayMana(cost string) Effect {
	return &sacrificeUnlessPayManaEffect{
		cost:   cost,
		prompt: "pay " + cost + " or sacrifice",
	}
}

func (e *sacrificeUnlessPayManaEffect) Text() string {
	return fmt.Sprintf("sacrifice unless you pay %s", e.cost)
}

func (e *sacrificeUnlessPayManaEffect) Properties() EffectProperties {
	return EffectProperties{}
}

func execSacrificeUnlessPayMana(ctx *EffectContext, e *sacrificeUnlessPayManaEffect) error {
	g := ctx.Game
	perm := g.FindPermanent(ctx.SourceID)
	if perm == nil {
		return nil
	}
	prompt := e.prompt
	if name := perm.Name(); name != "" {
		prompt = "pay " + e.cost + " to keep " + name
	}
	p := g.GetPlayer(ctx.Controller)
	if p != nil && p.ChooseMayAbility(prompt) && g.TryPayCostFromLands(ctx.Controller, e.cost) {
		return nil
	}
	g.Sacrifice(perm)
	return nil
}
