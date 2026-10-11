package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCurrentTurnControlFilter(n Node) (mage.PermanentFilter, error) {
	if _, err := symbol(n); err != nil {
		return mage.PermanentFilter{}, err
	}
	switch n.Name {
	case "NotControlledSinceBeginningOfTurn":
		return mage.Not(mage.ControlledSinceCurrentTurnStart), nil
	case "ControlledSinceBeginningOfTurn":
		return mage.ControlledSinceCurrentTurnStart, nil
	default:
		return mage.PermanentFilter{}, unsupported(n, "current turn control filter")
	}
}
