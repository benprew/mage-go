package ron

import "github.com/benprew/mage-go/pkg/mage"

type chosenDamageSourcePlan struct{ filter mage.DamageSourceFilter }

func compileExactSourcePrevention(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient" {
		return nil, unsupported(event, "exact-source prevention event")
	}
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	source := event.Args[0]
	if source.Kind != Symbol || source.Name != "TheChosenDamageSource" || ctx.chosenDamageSource == nil {
		return nil, problem(source, "unbound or incompatible chosen damage source")
	}
	recipient := event.Args[1]
	if recipient.Name != "Player" {
		return nil, unsupported(recipient, "exact-source prevention recipient")
	}
	if err := arity(recipient, 1); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(recipient.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Kind != Symbol || operations[0].Name != "PreventThatDamage" {
		return nil, problem(n.Args[1], "exact-source prevention requires the symbol PreventThatDamage")
	}
	plan := ctx.chosenDamageSource
	return func(bindings instanceBindings) mage.Effect {
		return mage.PreventNextDamageFromSelectedSource(mage.FromDamageSource(bindings.chosenDamageSources[plan]), player.build(bindings), plan.filter)
	}, nil
}
