package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileStaticCounterFilter(n Node) (mage.PermanentFilter, error) {
	count := 1
	if n.Name == "HasNumberCountersOfType" {
		count = 2
	}
	if err := arity(n, count); err != nil {
		return mage.PermanentFilter{}, err
	}
	counter, err := compilePermanentCounterType(n.Args[count-1])
	if err != nil {
		return mage.PermanentFilter{}, err
	}
	if count == 1 {
		return mage.HasCounter(counter), nil
	}
	comparison := n.Args[0]
	if comparison.Name != "EqualTo" {
		return mage.PermanentFilter{}, unsupported(comparison, "static counter comparison")
	}
	if err := arity(comparison, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	value := comparison.Args[0]
	if value.Name != "Integer" {
		return mage.PermanentFilter{}, unsupported(value, "static counter value")
	}
	if err := arity(value, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
		return mage.PermanentFilter{}, problem(value, "static counter count requires a nonnegative integer literal")
	}
	return mage.CounterCountEquals(counter, value.Args[0].Integer), nil
}
