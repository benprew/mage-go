package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSelectedSourceRedirect(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "NextTimeDamageWouldBeDealtThisTurnBySourceToRecipient" {
		return nil, unsupported(event, "selected-source redirection event")
	}
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	source := event.Args[0]
	if source.Kind != Symbol || source.Name != "TheChosenDamageSource" || ctx.chosenDamageSource == nil {
		return nil, problem(source, "unbound or incompatible chosen damage source")
	}
	recipient := event.Args[1]
	if recipient.Name != "Permanent" {
		return nil, unsupported(recipient, "selected-source redirection recipient")
	}
	if err := arity(recipient, 1); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(recipient.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "selected-source redirection requires exactly one operation")
	}
	operation := operations[0]
	if operation.Name != "DealToPlayerInstead" {
		return nil, unsupported(operation, "selected-source redirection operation")
	}
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(operation.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	plan := ctx.chosenDamageSource
	return func(bindings instanceBindings) mage.Effect {
		return mage.RedirectNextDamageFromSelectedSource(mage.FromDamageSource(bindings.chosenDamageSources[plan]), subject.build(bindings), player.build(bindings), plan.filter)
	}, nil
}
