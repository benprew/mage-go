package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileRandomBlockingPiles(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "random blocking piles require an ordinary resolving body")
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[0], "random blocking duration")
	}
	if n.Args[1].Kind != Symbol || n.Args[1].Name != "DefendingPlayersChooseCreaturesToDefendAttackersAtRandom" {
		return nil, unsupported(n.Args[1], "temporary game operation")
	}
	return func(instanceBindings) mage.Effect { return mage.RandomizeBlockingPilesThisTurn() }, nil
}
