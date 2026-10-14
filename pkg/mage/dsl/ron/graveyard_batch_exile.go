package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileGraveyardBatchExile(subject, modifiers Node, ctx scope) (effectFactory, error) {
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	zone := subject.Args[0]
	if zone.Name != "InAPlayersGraveyard" {
		return nil, unsupported(zone, "graveyard exile collection")
	}
	if err := arity(zone, 1); err != nil {
		return nil, err
	}
	player := zone.Args[0]
	if player.Name != "SinglePlayer" {
		return nil, unsupported(player, "graveyard exile player collection")
	}
	if err := arity(player, 1); err != nil {
		return nil, err
	}
	ref, err := compilePlayerRef(player.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	operations, err := list(modifiers)
	if err != nil {
		return nil, err
	}
	if len(operations) != 0 {
		return nil, unsupported(modifiers, "graveyard exile modifiers")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.ExileGraveyardFor(ref.build(bindings))
	}, nil
}
