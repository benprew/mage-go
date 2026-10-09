package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileActivationManaX(n Node, ctx scope) ([]mage.ActionOption, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if !ctx.x {
		return nil, problem(n, "XIs requires an X mana payment")
	}
	value := n.Args[0]
	if value.Name != "TheTotalNumberOfCountersOfTypeAmongPermanents" {
		return nil, unsupported(value, "source-counter activation X")
	}
	if err := arity(value, 2); err != nil {
		return nil, err
	}
	counter, err := compilePermanentCounterType(value.Args[0])
	if err != nil {
		return nil, err
	}
	subject := value.Args[1]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "activation X source")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	if subject.Args[0].Kind != Symbol || subject.Args[0].Name != "ThisPermanent" {
		return nil, unsupported(subject.Args[0], "activation X source")
	}
	return []mage.ActionOption{mage.WithActivationManaXFromSourceCounters(counter)}, nil
}

func containsDeclarationName(n Node, name string) bool { return countDeclarationName(n, name) != 0 }

func countDeclarationName(n Node, name string) int {
	count := 0
	if n.Name == name {
		count++
	}
	for _, child := range n.Args {
		count += countDeclarationName(child, name)
	}
	for _, child := range n.Fields {
		count += countDeclarationName(child, name)
	}
	return count
}
