package mage

import "slices"

// HasCreatureEnchantAbility checks the fixed declaration on this visit.
// Use CreatureEnchantTarget to declare this ability.
func (p *Permanent) HasCreatureEnchantAbility() bool {
	if p.PhasedOut || p.creatureEnchantRemoved || p.printedAbilitiesSuppressed {
		return false
	}
	return slices.ContainsFunc(p.Card.CastTargets(), isFixedCreatureEnchant)
}

func isFixedCreatureEnchant(target Target) bool {
	if binding, ok := target.(TargetBinding); ok {
		target = binding.targetSpec()
	}
	declaration, ok := UnwrapTarget(target).(*PermanentTarget)
	return ok && declaration.fixedCreatureEnchant
}

func targetsWithoutCreatureEnchant(targets []Target) []Target {
	kept := make([]Target, 0, len(targets))
	for _, target := range targets {
		if !isFixedCreatureEnchant(target) {
			kept = append(kept, target)
		}
	}
	return kept
}
