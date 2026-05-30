package fallen_empires

import . "github.com/benprew/mage-go/pkg/mage/dsl"

// withExpansion wraps a card factory (no-op until WithExpansion is added to the engine).
func withExpansion(factory func() Card) func() Card {
	return factory
}
