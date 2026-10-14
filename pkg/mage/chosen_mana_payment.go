package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type chosenManaPayment struct {
	player PlayerRef
	result *NumberBinding
}

// PayChosenManaInto saves a successful chosen mana payment at resolution.
// Zero or a failed payment clears the previous result.
func PayChosenManaInto(player PlayerRef, result *NumberBinding) ResolvingEffect {
	return &chosenManaPayment{player: player, result: result}
}
func (*chosenManaPayment) Text() string                 { return "pay any amount of mana" }
func (*chosenManaPayment) Properties() EffectProperties { return EffectProperties{} }
func (e *chosenManaPayment) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.numberBindings == nil {
		ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
	}
	ctx.Game.resolution.numberBindings[e.result] = 0
	id, ok := e.player.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(id) == nil {
		return nil
	}
	amount := chooseNonnegativeNumber(ctx.Game.GetPlayer(id), e.Text())
	if amount < 0 {
		return nil
	}
	if tryResolutionCostPayment(ctx, id, []Cost{&ManaCostPayment{MC: core.ManaCost{Generic: amount}}}, true) {
		ctx.Game.resolution.numberBindings[e.result] = amount
	}
	return nil
}
func (e *chosenManaPayment) inspectDefinition() definitionNode {
	n := definitionNode{Role: "chosen-mana-payment", Text: e.Text(), bindingWrite: e.result, children: []definitionEdge{{"player", e.player}, {"result", e.result}}}
	if e.result == nil {
		n.problem = "supply a paid mana binding"
	}
	return n
}
func (e *chosenManaPayment) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}
