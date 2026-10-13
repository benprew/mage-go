package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCounterPointPrevention(n Node, ctx scope) (optionFactory, error) {
	event := n.Args[0]
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	recipient := event.Args[0]
	if recipient.Name != "Permanent" {
		return nil, unsupported(recipient, "per-point prevention recipient")
	}
	if err := arity(recipient, 1); err != nil {
		return nil, err
	}
	checkSource := func(ref Node) error {
		plan, err := compilePermanentRef(ref, ctx)
		if err != nil {
			return err
		}
		if plan.kind != permanentSource {
			return unsupported(ref, "per-point prevention source")
		}
		return nil
	}
	if err := checkSource(recipient.Args[0]); err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Name != "If" {
		return nil, problem(n.Args[1], "per-point prevention requires one counter condition")
	}
	branch := operations[0]
	if err := arity(branch, 2); err != nil {
		return nil, err
	}
	condition := branch.Args[0]
	if condition.Name != "PermanentPassesFilter" {
		return nil, unsupported(condition, "per-point prevention condition")
	}
	if err := arity(condition, 2); err != nil {
		return nil, err
	}
	if err := checkSource(condition.Args[0]); err != nil {
		return nil, err
	}
	filter := condition.Args[1]
	if filter.Name != "HasACounterOfType" {
		return nil, unsupported(filter, "per-point prevention counter condition")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	kind, err := compilePermanentCounterType(filter.Args[0])
	if err != nil {
		return nil, err
	}
	body, err := list(branch.Args[1])
	if err != nil {
		return nil, err
	}
	if len(body) != 2 || body[0].Name != "RemoveCounters" || body[1].Kind != Symbol || body[1].Name != "PreventThatDamage" {
		return nil, problem(branch.Args[1], "per-point prevention requires counter removal then PreventThatDamage")
	}
	if err := arity(body[0], 1); err != nil {
		return nil, err
	}
	removals, err := list(body[0].Args[0])
	if err != nil {
		return nil, err
	}
	if len(removals) != 1 || removals[0].Name != "ACounterOfTypeFromPermanent" {
		return nil, problem(body[0], "per-point prevention requires one source counter removal")
	}
	removal := removals[0]
	if err := arity(removal, 2); err != nil {
		return nil, err
	}
	removedKind, err := compilePermanentCounterType(removal.Args[0])
	if err != nil {
		return nil, err
	}
	if removedKind != kind {
		return nil, problem(removal.Args[0], "removed counter must match the condition")
	}
	if err := checkSource(removal.Args[1]); err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		return mage.WithStaticAbility(mage.PreventDamageToSourceByRemovingCounters(kind))
	}, nil
}
