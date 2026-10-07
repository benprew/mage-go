package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileResolvingCounterFilter(n Node) (mage.PermanentFilter, error) {
	if n.Name == "HasACounterOfType" {
		return compileStaticCounterFilter(n)
	}
	if n.Name == "HasNoCountersOfType" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		counter, err := compilePermanentCounterType(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		return mage.CounterCountEquals(counter, 0), nil
	}
	if err := arity(n, 2); err != nil {
		return mage.PermanentFilter{}, err
	}
	comparison := n.Args[0]
	if comparison.Name == "EqualTo" {
		return compileStaticCounterFilter(n)
	}
	if comparison.Name != "GreaterThanOrEqualTo" {
		return mage.PermanentFilter{}, unsupported(comparison, "resolving counter comparison")
	}
	if err := arity(comparison, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	value := comparison.Args[0]
	if value.Name != "Integer" {
		return mage.PermanentFilter{}, unsupported(value, "resolving counter value")
	}
	if err := arity(value, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
		return mage.PermanentFilter{}, problem(value, "resolving counter count requires a nonnegative integer literal")
	}
	counter, err := compilePermanentCounterType(n.Args[1])
	if err != nil {
		return mage.PermanentFilter{}, err
	}
	return mage.CounterCountAtLeast(counter, value.Args[0].Integer), nil
}
