package mage

import "fmt"

// CoinBranch selects a result of a coin flip during resolution.
type CoinBranch interface {
	coinBranch()
	coinResult() (bool, ResolvingEffect)
}

type coinBranch struct {
	win    bool
	effect ResolvingEffect
}

func (coinBranch) coinBranch()                           {}
func (b coinBranch) coinResult() (bool, ResolvingEffect) { return b.win, b.effect }
func (b coinBranch) inspectDefinition() definitionNode {
	role := "on-loss"
	if b.win {
		role = "on-win"
	}
	return definitionNode{Role: role, children: []definitionEdge{{"effect", b.effect}}}
}

// OnWin runs the effect if the instructed player wins the flip.
func OnWin(effect ResolvingEffect) CoinBranch { return coinBranch{true, effect} }

// OnLoss runs the effect if the instructed player loses the flip.
func OnLoss(effect ResolvingEffect) CoinBranch { return coinBranch{false, effect} }

type coinFlipEffect struct {
	player   PlayerRef
	branches []CoinBranch
}

// FlipCoin makes one choice through the game's random source at resolution.
// An omitted result has no effect. Each supplied result must be unique.
func FlipCoin(player PlayerRef, branches ...CoinBranch) ResolvingEffect {
	return &coinFlipEffect{player, append([]CoinBranch(nil), branches...)}
}
func (*coinFlipEffect) Text() string { return "flip a coin" }
func (e *coinFlipEffect) Properties() EffectProperties {
	var effects []Effect
	for _, branch := range e.branches {
		if !nilDefinition(branch) {
			_, effect := branch.coinResult()
			if !nilDefinition(effect) {
				effects = append(effects, effect)
			}
		}
	}
	return CompositeEffects("", effects...).Properties()
}
func (e *coinFlipEffect) Apply(ctx *EffectContext) error {
	player, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	win := ctx.Game.FlipCoin(player)
	for _, branch := range e.branches {
		result, effect := branch.coinResult()
		if result == win {
			return effect.Apply(ctx)
		}
	}
	return nil
}
func (e *coinFlipEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "coin-flip", Text: e.Text(), children: []definitionEdge{{"player", e.player}}}
	if len(e.branches) == 0 {
		n.problem = "supply at least one coin branch"
	}
	seen := map[bool]bool{}
	for i, branch := range e.branches {
		if !nilDefinition(branch) {
			result, _ := branch.coinResult()
			if seen[result] {
				n.problem = "remove duplicate coin result branch"
			}
			seen[result] = true
		}
		n.children = append(n.children, definitionEdge{fmt.Sprintf("branches[%d]", i), branch})
	}
	return n
}
