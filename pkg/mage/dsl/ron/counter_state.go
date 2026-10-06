package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCounterStateTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 3); err != nil {
		return nil, err
	}
	subject := event.Args[0]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "counter-state subject")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(subject.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentSource {
		return nil, unsupported(subject.Args[0], "counter-state source")
	}
	comparison := event.Args[1]
	if comparison.Name != "GreaterThanOrEqualTo" {
		return nil, unsupported(comparison, "counter-state comparison")
	}
	if err := arity(comparison, 1); err != nil {
		return nil, err
	}
	value := comparison.Args[0]
	if value.Name != "Integer" {
		return nil, unsupported(value, "counter-state threshold")
	}
	if err := arity(value, 1); err != nil {
		return nil, err
	}
	literal := value.Args[0]
	if literal.Kind != IntegerNode || literal.Integer < 0 {
		return nil, problem(literal, "counter-state threshold requires a nonnegative integer literal")
	}
	counter, err := compilePermanentCounterType(event.Args[2])
	if err != nil {
		return nil, err
	}
	ctx.counterStateSource = true
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		actions := plan.build()
		trigger := mage.WhenState(mage.SourceHasCounters(counter, literal.Integer), mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}

func compileResolvingCounterRemoval(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	instructions, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(instructions) != 1 {
		return nil, problem(n, "counter removal requires one instruction")
	}
	instruction := instructions[0]
	if instruction.Name != "AllCountersOfTypeFromPermanent" && instruction.Name != "ACounterOfTypeFromPermanent" {
		return nil, unsupported(instruction, "counter removal instruction")
	}
	if err := arity(instruction, 2); err != nil {
		return nil, err
	}
	counter, err := compilePermanentCounterType(instruction.Args[0])
	if err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(instruction.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		if instruction.Name == "ACounterOfTypeFromPermanent" {
			return mage.RemoveCountersFrom(subject.build(bindings), counter, mage.Fixed(1))
		}
		return mage.RemoveAllCounters(subject.build(bindings), counter)
	}, nil
}
