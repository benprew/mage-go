package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileTemporaryCollection(n Node) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "duration")
	}
	subjects, err := compileStaticPermanentFilter(n.Args[0])
	if err != nil {
		return nil, err
	}
	if subjects.excludeSource {
		return nil, unsupported(n.Args[0], "temporary collection source exclusion")
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(n.Args[1], "supply at least one temporary modification")
	}
	type adjustment struct{ power, toughness int }
	adjustments := make([]adjustment, len(parts))
	for i, part := range parts {
		if part.Name != "AdjustPT" {
			return nil, unsupported(part, "temporary collection modification")
		}
		power, toughness, err := compilePT(part)
		if err != nil {
			return nil, err
		}
		adjustments[i] = adjustment{power, toughness}
	}
	return func(_ instanceBindings) mage.Effect {
		modifications := make([]mage.ContinuousModification, len(adjustments))
		for i, value := range adjustments {
			modifications[i] = mage.ModifyEachPT(subjects.build(), mage.Fixed(value.power), mage.Fixed(value.toughness))
		}
		return mage.Until(core.EndOfTurn, modifications...)
	}, nil
}
