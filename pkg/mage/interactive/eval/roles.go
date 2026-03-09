package eval

import (
	"github.com/mage/mage/pkg/mage"
	"github.com/mage/mage/pkg/mage/core"
)

// PermanentRole classifies a permanent's strategic function on the board.
type PermanentRole int

const (
	RoleThreat  PermanentRole = iota // power >= 3, or has evasion keywords
	RoleUtility                      // has tap abilities that produce non-mana effects
	RoleEngine                       // generates incremental advantage (triggered abilities with benefit)
	RoleMana                         // lands, mana creatures, mana artifacts
	RoleDefense                      // walls, high-toughness low-power blockers
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
func ClassifyPermanent(p *mage.Permanent) PermanentRole {
	isCreature := p.HasType(core.TypeCreature)
	isLand := p.HasType(core.TypeLand)

	if isLand {
		return RoleMana
	}
	if hasManaAbility(p) {
		return RoleMana
	}

	if !isCreature {
		if hasTriggeredAbilities(p) {
			return RoleEngine
		}
		return RoleUtility
	}

	if hasUtilityActivatedAbility(p) {
		return RoleUtility
	}

	pw := permPower(p)
	tg := permToughness(p)
	if p.HasKeyword(core.Defender) {
		return RoleDefense
	}
	if tg > 0 && pw >= 0 && tg > pw*2 {
		return RoleDefense
	}

	if pw >= 3 || hasEvasion(p) {
		return RoleThreat
	}

	return RoleThreat
}

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

func hasManaAbility(p *mage.Permanent) bool {
	for _, a := range p.RuntimeAbilities {
		inner := mage.UnwrapAbility(a)
		if _, ok := inner.(*mage.ManaAbility); ok {
			return true
		}
	}
	return false
}

func hasTriggeredAbilities(p *mage.Permanent) bool {
	for _, a := range p.RuntimeAbilities {
		inner := mage.UnwrapAbility(a)
		if _, ok := inner.(mage.TriggeredAbility); ok {
			return true
		}
	}
	return false
}

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

func boardDiversity(roles map[PermanentRole]int) int {
	distinctRoles := 0
	for _, count := range roles {
		if count > 0 {
			distinctRoles++
		}
	}
	if distinctRoles <= 1 {
		return 0
	}
	return (distinctRoles - 1) * 2
}
