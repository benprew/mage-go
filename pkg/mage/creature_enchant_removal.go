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
