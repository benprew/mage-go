package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileStaticDestructionReplacement(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "PermanentWouldBeDestroyed" {
		return nil, unsupported(event, "static destruction event")
	}
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	if event.Args[0].Kind != Symbol || event.Args[0].Name != "ThisPermanent" {
		return nil, unsupported(event.Args[0], "static destruction subject")
	}
	ops, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(ops) != 1 || ops[0].Name != "RegeneratePermanent" {
		return nil, problem(n.Args[1], "supply one regeneration operation")
	}
	if err := destructionEventSubject(ops[0]); err != nil {
		return nil, err
	}
	return func() mage.CardOption { return mage.WithAbility(mage.RegenerateInsteadOfDestroy()) }, nil
}
func compileFutureDestructionReplacement(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "NextTimePermanentWouldBeDestroyedThisTurn" {
		return nil, unsupported(event, "future destruction event")
	}
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(event.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	ops, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(ops) != 2 || ops[0].Kind != Symbol || ops[0].Name != "CancelDestroy" || ops[1].Name != "RemoveAllDamageFromPermanent" {
		return nil, problem(n.Args[1], "supply CancelDestroy and one damage removal operation")
	}
	if err := destructionEventSubject(ops[1]); err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.ReplaceNextDestructionThisTurn(subject.build(bindings))
	}, nil
}
func destructionEventSubject(n Node) error {
	if err := arity(n, 1); err != nil {
		return err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "WouldBeDestroyed_ThatPermanent" {
		return unsupported(n.Args[0], "destruction event subject")
	}
	return nil
}
