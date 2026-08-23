package eval

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// TargetPurpose describes why the AI is choosing a target.
type TargetPurpose int

const (
	TargetGeneric TargetPurpose = iota
	TargetRemoval
	TargetBurn
	TargetTap
	TargetPump
	TargetBounce
	TargetAura
	TargetCounters
	TargetExile
)

// TargetPurposeForEffects classifies target choice from effect metadata.
func TargetPurposeForEffects(effects []mage.Effect) TargetPurpose {
	purpose := TargetGeneric
	for _, e := range effects {
		props := e.Properties()
		if override := TargetPurposeFromAI(props.TargetPurposeOverride); override != TargetGeneric {
			return override
		}
		if props.IsBounce {
			return TargetBounce
		}
		if props.DamageValue != nil && props.Outcome == mage.OutcomeDetriment {
			return TargetBurn
		}
		if props.Taps && props.Outcome == mage.OutcomeDetriment {
			purpose = TargetTap
		}
		if props.Outcome == mage.OutcomeDetriment {
			if props.PowerBoost < 0 || props.ToughnessBoost < 0 {
				return TargetRemoval
			}
			if purpose == TargetGeneric {
				purpose = TargetRemoval
			}
		}
		if props.Outcome == mage.OutcomeBenefit {
			if props.PowerBoost != 0 || props.ToughnessBoost != 0 || props.GrantedKeyword != 0 {
				return TargetPump
			}
			purpose = TargetAura
		}
	}
	return purpose
}

func TargetPurposeFromAI(p mage.AITargetPurpose) TargetPurpose {
	switch p {
	case mage.AITargetRemoval:
		return TargetRemoval
	case mage.AITargetBurn:
		return TargetBurn
	case mage.AITargetTap:
		return TargetTap
	case mage.AITargetPump:
		return TargetPump
	case mage.AITargetBounce:
		return TargetBounce
	case mage.AITargetAura:
		return TargetAura
	case mage.AITargetCounters:
		return TargetCounters
	case mage.AITargetExile:
		return TargetExile
	default:
		return TargetGeneric
	}
}

// PermanentValueForTargeting scores a permanent as a target for the given
// purpose. Higher is better for spending that effect on the permanent.
func PermanentValueForTargeting(g *mage.Game, perm *mage.Permanent, purpose TargetPurpose) int {
	if perm == nil {
		return -1000
	}

	value := evalPermanentForTargeting(g, perm)
	switch purpose {
	case TargetRemoval:
		if perm.HasKeyword(core.Indestructible) {
			return -20
		}
		if perm.HasType(core.TypeCreature) && alreadyLethallyDamaged(g, perm) {
			return -15
		}
		if perm.Tapped && perm.HasAttr(core.AttrDoesNotUntap) {
			value /= 2
		}
		if perm.HasType(core.TypeCreature) {
			aurasAttached := 0
			for _, aura := range g.FilterBattlefield(mage.And(mage.IsEnchantment, mage.Not(mage.IsCreature))) {
				if aura.AttachedTo == perm.ID() {
					aurasAttached++
				}
			}
			value += aurasAttached * 4
		}
		return value
	case TargetExile:
		// Exile, sacrifice, and -X/-X remove indestructible creatures, so unlike
		// TargetRemoval there is no indestructibility penalty.
		if perm.HasType(core.TypeCreature) && alreadyLethallyDamaged(g, perm) {
			return -15
		}
		if perm.Tapped && perm.HasAttr(core.AttrDoesNotUntap) {
			value /= 2
		}
		if perm.HasType(core.TypeCreature) {
			aurasAttached := 0
			for _, aura := range g.FilterBattlefield(mage.And(mage.IsEnchantment, mage.Not(mage.IsCreature))) {
				if aura.AttachedTo == perm.ID() {
					aurasAttached++
				}
			}
			value += aurasAttached * 4
		}
		return value
	case TargetBurn:
		if perm.HasType(core.TypeCreature) && alreadyLethallyDamaged(g, perm) {
			return -15
		}
		return value
	case TargetBounce:
		if perm.IsToken {
			return value
		}
		if perm.HasType(core.TypeCreature) {
			aurasAttached := 0
			for _, aura := range g.FilterBattlefield(mage.And(mage.IsEnchantment, mage.Not(mage.IsCreature))) {
				if aura.AttachedTo == perm.ID() {
					aurasAttached++
				}
			}
			value += aurasAttached * 4
		}
		return value * 3 / 4
	case TargetTap:
		if !perm.HasType(core.TypeCreature) {
			return value / 3
		}
		if perm.Tapped {
			return -8
		}
		score := value
		if perm.CanDeclareAsAttacker(g) || perm.CanDeclareAsBlocker(g) {
			score += 5
		}
		if hasTargetingEvasion(perm) {
			score += 3
		}
		return score
	case TargetPump, TargetAura, TargetCounters:
		if !perm.HasType(core.TypeCreature) {
			return value / 2
		}
		power := max(perm.CurrentPower(g), 0)
		toughness := max(perm.CurrentToughness(g), 0)
		score := power*3 + toughness
		if perm.Tapped {
			score -= 2
		}
		if purpose == TargetPump && perm.HasAttr(core.AttrSummonSick) && !perm.HasAttr(core.Haste) {
			score -= 8
		}
		if perm.HasKeyword(core.Defender) || perm.HasAttr(core.AttrDoesNotUntap) {
			score -= 4
		}
		if hasTargetingEvasion(perm) {
			score += 3 + power
		}
		if perm.HasKeyword(core.DoubleStrike) {
			score += max(power, 1) * 2
		} else if perm.HasKeyword(core.FirstStrike) {
			score += max(power, 1)
		}
		if perm.HasKeyword(core.Lifelink) || perm.HasKeyword(core.Trample) {
			score += max(power, 1)
		}
		if perm.HasKeyword(core.Indestructible) || perm.HasKeyword(core.Hexproof) || perm.HasKeyword(core.Shroud) {
			score += 2
		}
		return score
	default:
		return value
	}
}

// TargetValueForPurpose scores either a permanent or player target.
func TargetValueForPurpose(g *mage.Game, controller, target uuid.UUID, purpose TargetPurpose, damage int) int {
	if perm := g.FindPermanent(target); perm != nil {
		score := PermanentValueForTargeting(g, perm, purpose)
		if purpose == TargetBurn && damage > 0 {
			toughnessLeft := perm.CurrentToughness(g) - perm.Damage
			if perm.ControllerID() != controller && toughnessLeft > 0 && damage >= toughnessLeft {
				score += 20
			} else {
				score = score/2 - 8
			}
		}
		return score
	}
	if tp := g.GetPlayer(target); tp != nil {
		if purpose != TargetBurn && purpose != TargetGeneric {
			return -10
		}
		score := 0
		if tp.PlayerID() != controller {
			score = 2
			if damage > 0 && damage >= tp.Life() {
				score = 1000
			} else if damage > 0 && tp.Life() <= damage+3 {
				score = 12
			}
		}
		return score
	}
	return -1000
}

func evalPermanentForTargeting(g *mage.Game, perm *mage.Permanent) int {
	if perm.HasType(core.TypeCreature) {
		return EvalCreatureInGame(perm, g)
	}
	return evalNonCreaturePermanent(perm, g)
}

func alreadyLethallyDamaged(g *mage.Game, perm *mage.Permanent) bool {
	toughness := perm.CurrentToughness(g)
	return toughness > 0 && perm.Damage >= toughness
}

func hasTargetingEvasion(perm *mage.Permanent) bool {
	return perm.HasKeyword(core.UnblockableKW) ||
		perm.HasKeyword(core.Flying) ||
		perm.HasKeyword(core.Fear) ||
		perm.HasKeyword(core.Menace) ||
		perm.HasKeyword(core.Islandwalk) ||
		perm.HasKeyword(core.Swampwalk) ||
		perm.HasKeyword(core.Forestwalk) ||
		perm.HasKeyword(core.Mountainwalk) ||
		perm.HasKeyword(core.Plainswalk)
}
