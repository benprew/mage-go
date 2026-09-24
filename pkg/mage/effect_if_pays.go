package mage

import (
	"fmt"
)

// ifPlayerPaysEffect is the resolution-time two-branch form of "a player may
// pay a cost" (CR 118.3). Declining, being unable to pay, or a failed payment
// selects the unpaid branch.
type ifPlayerPaysEffect struct {
	payer     PlayerSelector
	cost      Cost
	text      string
	prompt    string
	ifPaid    Effect
	ifNotPaid Effect
}

// IfPlayerPays asks exactly one selected player to pay cost, then resolves the
// corresponding branch in the existing EffectContext.
func IfPlayerPays(payer PlayerSelector, cost Cost, prompt string, ifPaid, ifNotPaid Effect) Effect {
	return newIfPlayerPays(payer, cost, prompt, prompt, ifPaid, ifNotPaid)
}

func newIfPlayerPays(payer PlayerSelector, cost Cost, text, prompt string, ifPaid, ifNotPaid Effect) Effect {
	return &ifPlayerPaysEffect{
		payer:     payer,
		cost:      cost,
		text:      text,
		prompt:    prompt,
		ifPaid:    ifPaid,
		ifNotPaid: ifNotPaid,
	}
}

func (e *ifPlayerPaysEffect) Text() string { return e.text }

func (e *ifPlayerPaysEffect) inspectDefinition() definitionNode {
	n := definitionNode{
		Role: "legacy-payment-branch", Text: e.Text(),
		children: []definitionEdge{{"payer", e.payer}, {"cost", e.cost}},
	}
	if e.ifPaid != nil {
		n.children = append(n.children, definitionEdge{"paid", e.ifPaid})
	}
	if e.ifNotPaid != nil {
		n.children = append(n.children, definitionEdge{"unpaid", e.ifNotPaid})
	}
	return n
}

func (e *ifPlayerPaysEffect) Properties() EffectProperties {
	var props EffectProperties
	if e.ifPaid != nil {
		mergeEffectProperties(&props, e.ifPaid.Properties())
	}
	if e.ifNotPaid != nil {
		mergeEffectProperties(&props, e.ifNotPaid.Properties())
	}
	return props
}

func (e *ifPlayerPaysEffect) Apply(ctx *EffectContext) error {
	g := ctx.Game
	if variable, ok := e.payer.(*VarPlayerSelector); ok {
		previous := variable.context
		variable.context = ctx
		defer func() { variable.context = previous }()
	}
	payerIDs := e.payer.Select(g, ctx.SourceID, ctx.Controller, ctx.Targets)
	if len(payerIDs) != 1 {
		return fmt.Errorf("expected exactly one payer, got %d", len(payerIDs))
	}
	pid := payerIDs[0]
	p := g.GetPlayer(pid)
	if p == nil {
		return ErrPlayerNotFound
	}
	paid := e.cost.CanPay(ctx.SourceID, pid, g) && p.ChooseMayAbility(e.prompt)
	if paid {
		paid = e.cost.Pay(ctx.SourceID, pid, g) == nil
	}
	if paid {
		if e.ifPaid == nil {
			return nil
		}
		return e.ifPaid.Apply(ctx)
	}
	if e.ifNotPaid == nil {
		return nil
	}
	return e.ifNotPaid.Apply(ctx)
}
