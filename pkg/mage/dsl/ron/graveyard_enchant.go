package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileEnchantGraveyardCard(n Node, ctx scope) (optionFactory, error) {
	if !ctx.aura {
		return nil, unsupported(n, "graveyard enchant rule without Aura subtype")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	filter, players := n.Args[0], n.Args[1]
	if filter.Name != "IsCardtype" {
		return nil, unsupported(filter, "graveyard enchant filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	if filter.Args[0].Kind != Symbol || filter.Args[0].Name != "Creature" {
		return nil, unsupported(filter, "graveyard enchant filter")
	}
	if players.Kind != Symbol || players.Name != "AnyPlayer" {
		return nil, unsupported(players, "graveyard enchant players")
	}
	return func() mage.CardOption { return mage.WithCastTarget(mage.GraveyardCreatureEnchantTarget()) }, nil
}
