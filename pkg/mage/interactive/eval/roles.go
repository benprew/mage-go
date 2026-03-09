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
// Uses a scoring system across multiple dimensions to pick the best-fitting role.
func ClassifyPermanent(p *mage.Permanent) PermanentRole {
	isCreature := p.HasType(core.TypeCreature)
	isLand := p.HasType(core.TypeLand)

	if isLand {
		return RoleMana
	}
	if hasManaAbility(p) {
		// Mana creatures with power >= 2 that can attack are partial threats.
		if isCreature && permPower(p) >= 2 && !p.HasKeyword(core.Defender) {
			return RoleThreat
		}
		return RoleMana
	}

	if !isCreature {
		// Non-creature permanents: check for engine (recurring advantage) vs utility.
		if hasTriggeredBenefitAbility(p) {
			return RoleEngine
		}
		if hasRepeatedActivatedAbility(p) {
			return RoleEngine
		}
		return RoleUtility
	}

	// Creatures: score each role dimension.
	pw := permPower(p)
	tg := permToughness(p)

	// Defense: Defender, or high toughness relative to power.
	if p.HasKeyword(core.Defender) {
		return RoleDefense
	}
	if tg > 0 && pw >= 0 && tg > pw*2 && pw <= 1 {
		return RoleDefense
	}

	// Engine: creatures with triggered benefit abilities (draw, tokens, lifegain).
	if hasTriggeredBenefitAbility(p) {
		return RoleEngine
	}

	// Utility: creatures with non-mana activated abilities.
	if hasUtilityActivatedAbility(p) {
		// High-power utility creatures are still threats first.
		if pw >= 3 || hasEvasion(p) {
			return RoleThreat
		}
		return RoleUtility
	}

	// Threat: power >= 3, evasion, double strike, or deathtouch.
	if pw >= 3 || hasEvasion(p) || p.HasKeyword(core.DoubleStrike) || p.HasKeyword(core.Deathtouch) {
		return RoleThreat
	}

	// Small creature with no special abilities — still a minor threat.
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

// hasTriggeredBenefitAbility returns true if the permanent has a triggered
// ability that generates incremental advantage (draw, tokens, etc.).
func hasTriggeredBenefitAbility(p *mage.Permanent) bool {
	for _, a := range p.RuntimeAbilities {
		inner := mage.UnwrapAbility(a)
		ta, ok := inner.(mage.TriggeredAbility)
		if !ok {
			continue
		}
		for _, e := range ta.Effects() {
			props := e.Properties()
			if props.DrawCount > 0 || props.Outcome == mage.OutcomeBenefit ||
				props.TokenPower > 0 || props.LifeGain > 0 {
				return true
			}
		}
	}
	return false
}

// hasRepeatedActivatedAbility returns true if the permanent has a non-mana
// activated ability with draw or benefit effects (card advantage engine).
func hasRepeatedActivatedAbility(p *mage.Permanent) bool {
	for _, a := range p.RuntimeAbilities {
		inner := mage.UnwrapAbility(a)
		aa, ok := inner.(mage.ActivatedAbility)
		if !ok {
			continue
		}
		for _, e := range aa.Effects() {
			props := e.Properties()
			if props.DrawCount > 0 {
				return true
			}
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
