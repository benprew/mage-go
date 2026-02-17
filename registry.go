package mage

import "fmt"

// CardFactory creates a new instance of a card.
type CardFactory func() Card

var registry = map[string]CardFactory{}

// Register adds a card factory to the global registry.
func Register(name string, factory CardFactory) {
	registry[name] = factory
}

// CreateCard creates a new card instance by name.
func CreateCard(name string) (Card, error) {
	f, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown card: %s", name)
	}
	return f(), nil
}

// CardRegistered returns true if a card with the given name is registered.
func CardRegistered(name string) bool {
	_, ok := registry[name]
	return ok
}
