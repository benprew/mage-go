package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileRepeatedEntry(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(n, "repeated entry requires a round")
	}
	round := parts[0]
	if round.Name != "EachPlayerStartingWithMayAction" {
		return nil, unsupported(round, "repeated round producer")
	}
	if len(parts) != 2 {
		return nil, problem(n, "repeated entry requires one round and its adjacent repeat condition")
	}
	continuation := parts[1]
	if err := arity(round, 3); err != nil {
		return nil, err
	}
	if round.Args[0].Kind != Symbol || round.Args[0].Name != "AnyPlayer" {
		return nil, unsupported(round.Args[0], "repeated entry player collection")
	}
	start, err := compilePlayerRef(round.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	entry := round.Args[2]
	if entry.Name != "PutACardFromHandOnBattlefield" {
		return nil, unsupported(entry, "repeated optional action")
	}
	filter, err := compileOptionalHandEntryFilter(entry)
	if err != nil {
		return nil, err
	}
	if continuation.Name != "Unless" {
		return nil, unsupported(continuation, "repeat continuation")
	}
	if err := arity(continuation, 2); err != nil {
		return nil, err
	}
	condition := continuation.Args[0]
	if condition.Kind != Symbol || condition.Name != "NoOneTookAnActionThisWay" {
		return nil, unsupported(condition, "repeated entry result condition")
	}
	again, err := list(continuation.Args[1])
	if err != nil {
		return nil, err
	}
	if len(again) != 1 || again[0].Kind != Symbol || again[0].Name != "RepeatThisProcess" {
		return nil, problem(continuation, "repeat continuation requires one bare RepeatThisProcess")
	}
	return func(bindings instanceBindings) mage.Effect {
		result := mage.BindNumber("repeated round entries at " + n.Position.String())
		body := mage.EachPlayerMayPutFromHandOntoBattlefield(mage.EachPlayer, start.build(bindings), filter, result)
		return mage.RepeatWhilePositive(body, mage.FromNumber(result))
	}, nil
}
