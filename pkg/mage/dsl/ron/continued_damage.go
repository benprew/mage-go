package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileContinuedDamage(n Node, ctx scope) (effectFactory, error) {
	event := n.Args[0]
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	source := event.Args[0]
	if source.Kind != Symbol || source.Name != "TheChosenDamageSource" || ctx.chosenDamageSource == nil {
		return nil, problem(source, "unbound or incompatible chosen damage source")
	}
	if !ctx.resolving || ctx.permanent || ctx.sourceZone != nil || ctx.enteringSource || ctx.drawReplacementBody || ctx.delayedStep || ctx.discardCauseEvent != nil {
		return nil, problem(n, "continued damage requires an ordinary spell resolving body")
	}
	recipient := event.Args[1]
	if recipient.Name != "Player" {
		return nil, unsupported(recipient, "continued damage recipient")
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
	if len(operations) != 2 {
		return nil, problem(n.Args[1], "continued damage requires continuation and one spell damage operation")
	}
	if operations[0].Kind != Symbol || operations[0].Name != "ContinueDealingDamage" {
		return nil, unsupported(operations[0], "continued damage operation")
	}
	damage := operations[1]
	if damage.Name != "SpellDealsDamage" {
		return nil, unsupported(damage, "continued damage operation")
	}
	if err := arity(damage, 3); err != nil {
		return nil, err
	}
	spell, err := compileSelfDamageSource(damage.Args[0], ctx, false)
	if err != nil {
		return nil, err
	}
	if damage.Args[1].Kind != Symbol || damage.Args[1].Name != "WouldDealDamage_ThatMuchDamage" {
		return nil, unsupported(damage.Args[1], "continued proposed damage amount")
	}
	destination := damage.Args[2]
	if destination.Name != "Player" {
		return nil, unsupported(destination, "continued damage destination")
	}
	if err := arity(destination, 1); err != nil {
		return nil, err
	}
	controller := destination.Args[0]
	if controller.Kind != Symbol || controller.Name != "WouldDealDamage_ControllerOfDamageSource" {
		return nil, unsupported(controller, "proposed damage source controller")
	}
	plan := ctx.chosenDamageSource
	return func(bindings instanceBindings) mage.Effect {
		return mage.AddSpellDamageToNextSelectedSourceEvent(mage.FromDamageSource(bindings.chosenDamageSources[plan]), player.build(bindings), spell(bindings), plan.filter)
	}, nil
}
