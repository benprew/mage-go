package ron

func compileCounterAction(n Node, ctx scope) (effectFactory, error) {
	switch n.Name {
	case "PutCounters":
		return compileCounterPlacement(n, ctx)
	case "PutCountersWithRestriction":
		return compileBoundedCounterPlacement(n, ctx)
	case "RemoveCounters":
		return compileResolvingCounterRemoval(n, ctx)
	default:
		return nil, unsupported(n, "counter action")
	}
}
