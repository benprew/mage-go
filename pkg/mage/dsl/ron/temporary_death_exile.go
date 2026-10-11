package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileTemporaryDeathExile(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "APermanentWouldDie" {
		return nil, unsupported(event, "temporary death replacement event")
	}
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	subject := event.Args[0]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "temporary death replacement subject")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(subject.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	ops, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(ops) != 1 || ops[0].Kind != Symbol || ops[0].Name != "ExileItInstead" {
		return nil, problem(n.Args[1], "supply one ExileItInstead symbol")
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "temporary death replacement duration")
	}
	return func(bindings instanceBindings) mage.Effect { return mage.ExileIfWouldDieThisTurn(ref.build(bindings)) }, nil
}
