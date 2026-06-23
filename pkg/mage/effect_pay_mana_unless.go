package mage

import (
	"fmt"
)

// payManaUnlessEffect implements an OPTIONAL "do X unless you pay {cost}" branch
// (CR 118.3 / 603.4). The controller is asked whether to pay; only if they
// accept AND the mana can be paid from untapped lands is the ifNotPaid branch
// skipped. If they decline or cannot pay, ifNotPaid resolves.
//
// The player keeps the choice of taking the consequence even when they could
// afford the cost (e.g. paying Junún Efreet's upkeep, or Hasran Ogress's
// attack tax).
type payManaUnlessEffect struct {
	cost       string
	basePrompt string
	keepName   bool
	ifNotPaid  Effect
}

// SacrificeUnlessPayMana wraps an upkeep trigger so the controller may pay the
// given mana cost to keep the source. Declining (or being unable to pay)
// sacrifices the source. The prompt names the source via Player.ChooseMayAbility.
func SacrificeUnlessPayMana(cost string) Effect {
	return &payManaUnlessEffect{
		cost:       cost,
		basePrompt: "pay " + cost + " or sacrifice",
		keepName:   true,
		ifNotPaid:  SacrificeSourceStep(),
	}
}

// DamageUnlessPayMana lets the controller pay the given mana cost to avoid the
// source dealing amount damage to them. Declining (or being unable to pay)
// deals the damage. Used by cards like Hasran Ogress.
func DamageUnlessPayMana(cost string, amount int) Effect {
	return &payManaUnlessEffect{
		cost:       cost,
		basePrompt: fmt.Sprintf("pay %s or take %d damage", cost, amount),
		ifNotPaid:  DealDamageToPlayers(Fixed(amount), SelectController()),
	}
}

func (e *payManaUnlessEffect) Text() string {
	return fmt.Sprintf("%s unless you pay %s", e.ifNotPaid.Text(), e.cost)
}

func (e *payManaUnlessEffect) Properties() EffectProperties {
	return e.ifNotPaid.Properties()
}

func (e *payManaUnlessEffect) Apply(ctx *EffectContext) error {
	g := ctx.Game
	prompt := e.basePrompt
	if e.keepName {
		if perm := g.FindPermanent(ctx.SourceID); perm != nil && perm.Name() != "" {
			prompt = "pay " + e.cost + " to keep " + perm.Name()
		}
	}
	p := g.GetPlayer(ctx.Controller)
	if p != nil && p.ChooseMayAbility(prompt) && g.TryPayCostFromLands(ctx.Controller, e.cost) {
		return nil
	}
	return ApplyEffect(g, e.ifNotPaid, ctx.SourceID, ctx.Controller, ctx.Targets)
}
