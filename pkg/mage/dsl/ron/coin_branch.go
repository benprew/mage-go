package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCoinBranch(n Node, ctx scope) (effectFactory, error) {
	count := 1
	if n.Name == "FlipACoin_OnWinAndLose" {
		count = 2
	}
	if err := arity(n, count); err != nil {
		return nil, err
	}
	branches := make([][]effectFactory, count)
	for i, branch := range n.Args {
		parts, err := list(branch)
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(branch, "coin branch requires at least one action")
		}
		branches[i], err = compileActionSequence(parts, ctx)
		if err != nil {
			return nil, err
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		results := make([]mage.CoinBranch, count)
		for i, branch := range branches {
			effects := make([]mage.Effect, len(branch))
			for j, build := range branch {
				effects[j] = build(bindings)
			}
			body := mage.Sequence(effects...)
			if count == 2 && i == 0 {
				results[i] = mage.OnWin(body)
			} else {
				results[i] = mage.OnLoss(body)
			}
		}
		return mage.FlipCoin(mage.You, results...)
	}, nil
}
