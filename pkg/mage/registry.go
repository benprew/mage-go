package mage

import (
	"fmt"
	"strings"
	"sync"
	"uuid"
)

// CardFactory creates a new instance of a card.
type CardFactory func() *Card

// Registry holds immutable definitions and legacy factories with thread-safe access.
// folded maps a diacritic-stripped name (e.g. "El-Hajjaj") to the
// canonical name (e.g. "El-Hajjâj") so external callers using ASCII-only
// spellings (XMage's SetCardInfo, plain-text deck files) still resolve.
type Registry struct {
	mu          sync.RWMutex
	factories   map[string]CardFactory
	definitions map[string]*CardDefinition
	folded      map[string]string
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		factories:   make(map[string]CardFactory),
		definitions: make(map[string]*CardDefinition),
		folded:      make(map[string]string),
	}
}

// Register adds a card factory to this registry.
func (r *Registry) Register(name string, factory CardFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.definitions, name)
	r.factories[name] = factory
	if folded := FoldDiacritics(name); folded != name {
		r.folded[folded] = name
	}
}

// CreateCard creates a new card instance by name from this registry.
func (r *Registry) CreateCard(name string) (*Card, error) {
	r.mu.RLock()
	canonical := name
	if r.definitions[name] == nil && r.factories[name] == nil {
		if mapped, ok := r.folded[name]; ok {
			canonical = mapped
		}
	}
	definition := r.definitions[canonical]
	factory := r.factories[canonical]
	r.mu.RUnlock()
	if definition != nil {
		return definition.NewInstance(uuid.Nil()), nil
	}
	if factory == nil {
		return nil, fmt.Errorf("unknown card: %s", name)
	}
	return factory(), nil
}

// RegisterDefinition publishes a definition without caching a mutable card prototype.
func (r *Registry) RegisterDefinition(definition *CardDefinition) error {
	return r.RegisterDefinitions(definition)
}

// RegisterDefinitions checks the complete batch before it publishes any definition.
func (r *Registry) RegisterDefinitions(definitions ...*CardDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	names := make(map[string]string, len(r.factories)+len(r.definitions)+len(definitions))
	for name := range r.factories {
		names[FoldDiacritics(name)] = name
	}
	for name := range r.definitions {
		names[FoldDiacritics(name)] = name
	}
	for _, definition := range definitions {
		if definition == nil || definition.Name() == "" || len(definition.printed.TypeLine.CardTypes) == 0 {
			return fmt.Errorf("invalid card definition")
		}
		name := definition.Name()
		folded := FoldDiacritics(name)
		if existing, ok := names[folded]; ok {
			return fmt.Errorf("card already registered: %s", existing)
		}
		names[folded] = name
	}
	for _, definition := range definitions {
		name := definition.Name()
		r.definitions[name] = definition
		if folded := FoldDiacritics(name); folded != name {
			r.folded[folded] = name
		}
	}
	return nil
}

// CreateInstance creates an owned instance from a registered definition.
// Legacy factories remain available through CreateCard.
func (r *Registry) CreateInstance(name string, owner uuid.UUID) (*Card, error) {
	r.mu.RLock()
	if r.definitions[name] == nil {
		if canonical, ok := r.folded[name]; ok {
			name = canonical
		}
	}
	definition := r.definitions[name]
	r.mu.RUnlock()
	if definition == nil {
		return nil, fmt.Errorf("no card definition registered: %s", name)
	}
	return definition.NewInstance(owner), nil
}

// foldDiacritics returns name with combining marks stripped, e.g.
// "El-Hajjâj" -> "El-Hajjaj". Used to map ASCII spellings of card
// names (XMage SetCardInfo, plain-text deck files) to their canonical
// Scryfall accented form. Covers the diacritic set actually used in MTG
// card names.
var diacriticReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ä", "a", "ã", "a", "å", "a", "Á", "A", "À", "A", "Â", "A", "Ä", "A", "Ã", "A", "Å", "A",
	"é", "e", "è", "e", "ê", "e", "ë", "e", "É", "E", "È", "E", "Ê", "E", "Ë", "E",
	"í", "i", "ì", "i", "î", "i", "ï", "i", "Í", "I", "Ì", "I", "Î", "I", "Ï", "I",
	"ó", "o", "ò", "o", "ô", "o", "ö", "o", "õ", "o", "Ó", "O", "Ò", "O", "Ô", "O", "Ö", "O", "Õ", "O",
	"ú", "u", "ù", "u", "û", "u", "ü", "u", "Ú", "U", "Ù", "U", "Û", "U", "Ü", "U",
	"ñ", "n", "Ñ", "N",
	"ç", "c", "Ç", "C",
	"ý", "y", "ÿ", "y", "Ý", "Y",
)

func FoldDiacritics(name string) string {
	return diacriticReplacer.Replace(name)
}

// CardRegistered returns true if a card with the given name is in this registry.
func (r *Registry) CardRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, factory := r.factories[name]
	_, definition := r.definitions[name]
	return factory || definition
}

// DefaultRegistry is the process-wide card registry used by all card init() functions.
var DefaultRegistry = NewRegistry()

// Register adds a card factory to the default registry.
func Register(name string, factory CardFactory) {
	DefaultRegistry.Register(name, factory)
}

// CreateCard creates a new card instance by name from the default registry.
func CreateCard(name string) (*Card, error) {
	return DefaultRegistry.CreateCard(name)
}

// CardRegistered returns true if a card with the given name is in the default registry.
func CardRegistered(name string) bool {
	return DefaultRegistry.CardRegistered(name)
}

// RegisteredCardNames returns a list of all registered card names.
func RegisteredCardNames() []string {
	return DefaultRegistry.RegisteredCardNames()
}

// RegisteredCardNames returns a list of all registered card names.
func (r *Registry) RegisteredCardNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.factories)+len(r.definitions))
	for name := range r.factories {
		names = append(names, name)
	}
	for name := range r.definitions {
		names = append(names, name)
	}
	return names
}

// RegisterDefinition adds an immutable definition to the default registry.
func RegisterDefinition(definition *CardDefinition) error {
	return DefaultRegistry.RegisterDefinition(definition)
}

// CreateInstance creates an owned instance from a definition in the default registry.
func CreateInstance(name string, owner uuid.UUID) (*Card, error) {
	return DefaultRegistry.CreateInstance(name, owner)
}
