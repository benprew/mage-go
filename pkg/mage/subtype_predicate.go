package mage

import (
	"fmt"
	"strings"
)

// ValidateSubtypePredicate checks a subtype name before a filter is built.
// Creature names remain open, as in the shared subtype layer service.
// Other families require a known name so a creature name cannot become a land type.
func ValidateSubtypePredicate(family SubtypeFamily, subtype string) error {
	if err := ValidateSubtypeFamily(family, subtype); err != nil {
		return err
	}
	if strings.TrimSpace(subtype) != subtype {
		return fmt.Errorf("subtype must not have outer spaces")
	}
	if family != SubtypeCreature {
		known, ok := knownSubtypeFamily(subtype)
		if !ok || known != family {
			return fmt.Errorf("unknown subtype for family: %s", subtype)
		}
	}
	return nil
}
