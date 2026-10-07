package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileGroupedCombatTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	parts, err := list(event.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 2 {
		return nil, problem(event, "grouped combat union requires both directions")
	}
	first, second := parts[0], parts[1]
	if first.Name == "WhenACreatureBecomesBlockedByAnyNumberOfCreatures" {
		first, second = second, first
	}
	if first.Name != "WhenACreatureBlocksAnyNumberOfCreatures" || second.Name != "WhenACreatureBecomesBlockedByAnyNumberOfCreatures" {
		return nil, unsupported(event, "grouped combat union")
	}
	for _, part := range []Node{first, second} {
		if err := arity(part, 2); err != nil {
			return nil, err
		}
	}
	if definitionKey(first.Args[0]) != definitionKey(second.Args[0]) || definitionKey(first.Args[1]) != definitionKey(second.Args[1]) {
		return nil, problem(event, "grouped combat union requires the same subject and filter")
	}
	subject := first.Args[0]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "grouped combat subject")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(subject.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentSource {
		return nil, unsupported(subject.Args[0], "grouped combat source")
	}
	filter := first.Args[1]
	if filter.Name != "IsCreatureType" {
		return nil, unsupported(filter, "grouped combat filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	subtype, err := symbol(filter.Args[0])
	if err != nil {
		return nil, err
	}
	subtype = subtypeName(subtype)
	if err := mage.ValidateSubtypeFamily(mage.SubtypeCreature, subtype); err != nil {
		return nil, problem(filter, err.Error())
	}
	ctx.sourceBlock = true
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		actions := plan.build()
		pattern := mage.BlocksOrBecomesBlockedBy(mage.This).OfSubtype(subtype)
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
