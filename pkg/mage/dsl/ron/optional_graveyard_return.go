package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileOptionalSourceReturn(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	card, err := compileCardRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if card.sourceZone == nil || card.sourceZone.zone != core.ZoneGraveyard || card.sourceZone.aura {
		return nil, problem(n.Args[0], "optional source return requires a non-Aura graveyard source card")
	}
	entries, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(entries) != 1 || entries[0].Kind != Symbol || entries[0].Name != "EntersNormally" {
		return nil, unsupported(n.Args[1], "optional source battlefield entry")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.MayReturnNonAuraCardToBattlefield(card.build(bindings), ctx.player.build(bindings))
	}, nil
}
