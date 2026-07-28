package mage

import "fmt"

// UnlessTargetPays creates an effect that prompts the selected player to
// pay the given cost. If they decline or cannot pay, ifNotPaid resolves;
// otherwise nothing further happens. Use SelectTargetPlayer() to make the
// targeted player the payer, SelectEachOpponent() for the controller's
// opponent, etc. The cost is paid via Game's standard cost machinery
// (it's not a CostFunc — pass any Cost, e.g. ManaCostOf("{3}") or
// LifeCost(2)).
func UnlessTargetPays(payer PlayerSelector, cost Cost, prompt string, ifNotPaid Effect) Effect {
	text := fmt.Sprintf("%s unless %s pays %s", ifNotPaid.Text(), payer.Text(), cost.Text())
	return newIfPlayerPays(payer, cost, text, prompt, nil, ifNotPaid)
}
