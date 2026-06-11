package mage

import (
	"fmt"
)

// effectIfPaid is the resolution-time "you may pay a cost. If you do, do X"
// branch (CR 118.3).
type effectIfPaid struct {
	cost   Cost
	effect Effect
	payer  PlayerSelector
}

// EffectIfPaid creates an effect that prompts the controller to pay the given
// cost. If they accept and payment succeeds, effect resolves.
func EffectIfPaid(cost Cost, effect Effect) Effect {
	return &effectIfPaid{
		cost:   cost,
		effect: effect,
		payer:  SelectController(),
	}
}

func (e *effectIfPaid) Text() string {
	return fmt.Sprintf("%s if you %s", e.effect.Text(), e.cost.Text())
}

func (e *effectIfPaid) Properties() EffectProperties {
	return e.effect.Properties()
}

func execIfPaid(ctx *EffectContext, e *effectIfPaid) error {
	g := ctx.Game
	payerIDs := e.payer.Select(g, ctx.SourceID, ctx.Controller, ctx.Targets)
	if len(payerIDs) > 1 || len(payerIDs) == 0 {
		return fmt.Errorf("ERROR: Wrong number of payers: %v", payerIDs)
	}
	pid := payerIDs[0]
	p := g.GetPlayer(pid)
	if p == nil {
		return fmt.Errorf("ERROR: No player for payer: %d", pid)
	}
	if !e.cost.CanPay(ctx.SourceID, pid, g) || !p.ChooseMayAbility(e.Text()) {
		return nil
	}
	if err := e.cost.Pay(ctx.SourceID, pid, g); err != nil {
		return fmt.Errorf("ERROR: Unable to pay for cost: %v", err)
	}
	return ApplyEffect(g, e.effect, ctx.SourceID, ctx.Controller, ctx.Targets)
}
