package mage

import (
	"fmt"
	"sync"
)

// CardFactory creates a new instance of a card.
type CardFactory func() Card

// Registry holds card factories keyed by name, with thread-safe access.
type Registry struct {
	mu       sync.RWMutex
	factories map[string]CardFactory
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]CardFactory),
	}
}

// Register adds a card factory to this registry.
func (r *Registry) Register(name string, factory CardFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// CreateCard creates a new card instance by name from this registry.
func (r *Registry) CreateCard(name string) (Card, error) {
	r.mu.RLock()
	f, ok := r.factories[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown card: %s", name)
	}
	return f(), nil
}

// CardRegistered returns true if a card with the given name is in this registry.
func (r *Registry) CardRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.factories[name]
	return ok
}

// DefaultRegistry is the process-wide card registry used by all card init() functions.
var DefaultRegistry = NewRegistry()

// Register adds a card factory to the default registry.
func Register(name string, factory CardFactory) {
	DefaultRegistry.Register(name, factory)
}

// CreateCard creates a new card instance by name from the default registry.
func CreateCard(name string) (Card, error) {
	return DefaultRegistry.CreateCard(name)
}

// CardRegistered returns true if a card with the given name is in the default registry.
func CardRegistered(name string) bool {
	return DefaultRegistry.CardRegistered(name)
}
