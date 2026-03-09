package interactive

import (
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// PermanentRole classifies a permanent's strategic function on the board.
type PermanentRole int

const (
	RoleThreat  PermanentRole = iota // power >= 3, or has evasion keywords
	RoleUtility                      // has tap abilities that produce non-mana effects (draw, removal, damage)
	RoleEngine                       // generates incremental advantage (triggered abilities with benefit)
	RoleMana                         // lands, mana creatures, mana artifacts
	RoleDefense                      // walls, high-toughness low-power blockers (toughness > power*2)
)

// String returns a human-readable name for the role.
func (r PermanentRole) String() string {
	switch r {
	case RoleThreat:
		return "Threat"
	case RoleUtility:
		return "Utility"
	case RoleEngine:
		return "Engine"
	case RoleMana:
		return "Mana"
	case RoleDefense:
		return "Defense"
	default:
		return "Unknown"
	}
}

// ClassifyPermanent determines the primary strategic role of a permanent.
// The classification checks in priority order: Mana > Utility > Engine > Defense > Threat.
// Non-creature, non-land permanents with triggered abilities are classified as Engines;
// non-creature, non-land permanents without special abilities default to Utility.
func ClassifyPermanent(p *mage.Permanent) PermanentRole {
	isCreature := p.HasType(core.TypeCreature)
	isLand := p.HasType(core.TypeLand)

	// Mana sources: lands, creatures with mana abilities, artifacts with mana abilities.
	if isLand {
		return RoleMana
	}
	if hasManaAbility(p) {
		return RoleMana
	}

	// Non-creature permanents: check for engine or utility.
	if !isCreature {
		if hasTriggeredAbilities(p) {
			return RoleEngine
		}
		return RoleUtility
	}

	// Creatures: check for utility (tap abilities with non-mana effects).
	if hasUtilityActivatedAbility(p) {
		return RoleUtility
	}

	// Defense: high toughness relative to power, or has Defender.
	pw := permPower(p)
	tg := permToughness(p)
	if p.HasKeyword(core.Defender) {
		return RoleDefense
	}
	if tg > 0 && pw >= 0 && tg > pw*2 {
		return RoleDefense
	}

	// Default creature: Threat (power >= 3, or has evasion).
	if pw >= 3 || hasEvasion(p) {
		return RoleThreat
	}

	// Small creatures without evasion: still threats (they attack).
	return RoleThreat
}

// hasEvasion returns true if the permanent has any evasion keyword.
func hasEvasion(p *mage.Permanent) bool {
	evasionKWs := []core.Keyword{
		core.Flying, core.Fear, core.Menace, core.UnblockableKW,
		core.Islandwalk, core.Swampwalk, core.Forestwalk,
		core.Mountainwalk, core.Plainswalk, core.Trample,
	}
	for _, kw := range evasionKWs {
		if p.HasKeyword(kw) {
			return true
		}
	}
	return false
}

// hasManaAbility returns true if the permanent has a mana ability.
func hasManaAbility(p *mage.Permanent) bool {
	for _, a := range p.RuntimeAbilities {
		inner := mage.UnwrapAbility(a)
		if _, ok := inner.(*mage.ManaAbility); ok {
			return true
		}
	}
	return false
}

// hasTriggeredAbilities returns true if the permanent has any triggered abilities.
func hasTriggeredAbilities(p *mage.Permanent) bool {
	for _, a := range p.RuntimeAbilities {
		inner := mage.UnwrapAbility(a)
		if _, ok := inner.(mage.TriggeredAbility); ok {
			return true
		}
	}
	return false
}

// hasUtilityActivatedAbility returns true if the permanent has a non-mana activated ability.
func hasUtilityActivatedAbility(p *mage.Permanent) bool {
	for _, a := range p.RuntimeAbilities {
		inner := mage.UnwrapAbility(a)
		switch inner.(type) {
		case *mage.ManaAbility:
			continue
		case mage.ActivatedAbility:
			return true
		}
	}
	return false
}

// countRoles counts permanents by role for a given controller.
func countRoles(perms []*mage.Permanent, controllerFilter func(*mage.Permanent) bool) map[PermanentRole]int {
	counts := make(map[PermanentRole]int)
	for _, p := range perms {
		if controllerFilter(p) {
			role := ClassifyPermanent(p)
			counts[role]++
		}
	}
	return counts
}

// boardDiversity returns a bonus for having permanents in multiple roles.
// Having both threats and utility is better than all threats.
func boardDiversity(roles map[PermanentRole]int) int {
	distinctRoles := 0
	for _, count := range roles {
		if count > 0 {
			distinctRoles++
		}
	}
	// Bonus for each distinct role beyond the first.
	if distinctRoles <= 1 {
		return 0
	}
	return (distinctRoles - 1) * 2
}
