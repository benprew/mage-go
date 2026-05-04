package mage

import (
	"fmt"

	"github.com/google/uuid"
)

// unlessPaysEffect is the resolution-time "do X unless that player pays Y"
// branch (CR 118.3). At resolution it asks the selected player whether to
// pay the cost; if they decline (or can't pay), the inner effect resolves.
// If they pay, the inner effect is skipped.
//
// This is NOT an additional cost — it's a per-card resolution-time branch
// that runs after the spell or ability has begun resolving. Used by spells
// like "Target opponent loses 4 life unless they pay {3}" and aura effects
// like "At the beginning of your draw step, you may draw a card. If you
// do, that opponent may pay {2}; if they do, you don't draw."
type unlessPaysEffect struct {
	payer       PlayerSelector
	cost        Cost
	prompt      string
	ifNotPaid   Effect
}

// UnlessTargetPays creates an effect that prompts the selected player to
// pay the given cost. If they decline or cannot pay, ifNotPaid resolves;
// otherwise nothing further happens. Use SelectTargetPlayer() to make the
// targeted player the payer, SelectEachOpponent() for the controller's
// opponent, etc. The cost is paid via Game's standard cost machinery
// (it's not a CostFunc — pass any Cost, e.g. ManaCostOf("{3}") or
// LifeCost(2)).
func UnlessTargetPays(payer PlayerSelector, cost Cost, prompt string, ifNotPaid Effect) Effect {
	return &unlessPaysEffect{
		payer:     payer,
		cost:      cost,
		prompt:    prompt,
		ifNotPaid: ifNotPaid,
	}
}

func (e *unlessPaysEffect) Text() string {
	return fmt.Sprintf("%s unless %s pays %s", e.ifNotPaid.Text(), e.payer.Text(), e.cost.Text())
}

func (e *unlessPaysEffect) Properties() EffectProperties {
	return e.ifNotPaid.Properties()
}

func (e *unlessPaysEffect) Apply(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
	payerIDs := e.payer.Select(g, sourceID, controller, targets)
	for _, pid := range payerIDs {
		// If the chosen payer can't pay at all, the cost is unpayable —
		// CR 118.3 / 118.4 say they "may pay"; an unpayable cost is the
		// same as declining, so we run ifNotPaid.
		if !e.cost.CanPay(sourceID, pid, g) {
			if err := e.ifNotPaid.Apply(g, sourceID, controller, targets); err != nil {
				return err
			}
			continue
		}
		p := g.GetPlayer(pid)
		if p == nil {
			continue
		}
		if !p.ChooseMayAbility(e.prompt) {
			if err := e.ifNotPaid.Apply(g, sourceID, controller, targets); err != nil {
				return err
			}
			continue
		}
		if err := e.cost.Pay(sourceID, pid, g); err != nil {
			// Payment unexpectedly failed — fall back to the not-paid branch.
			if err2 := e.ifNotPaid.Apply(g, sourceID, controller, targets); err2 != nil {
				return err2
			}
		}
	}
	return nil
}
