package combatsolver

import (
	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// CombatRole tags a card or activated ability with the kind of combat trick
// it represents. Used by the heuristic strategy to decide whether to hold
// the card for the post-blockers response window vs cast it sorcery-speed.
type CombatRole int

const (
	RoleNone       CombatRole = iota
	RolePump                  // +X/+Y until end of turn — hold for blockers/damage step
	RoleProtection            // hexproof / indestructible — hold for removal on stack
	RoleRemoval               // kill an attacker/blocker — hold to ambush
	RoleDamage                // direct damage to a creature
)

// String renders a CombatRole for diagnostics.
func (r CombatRole) String() string {
	switch r {
	case RolePump:
		return "Pump"
	case RoleProtection:
		return "Protection"
	case RoleRemoval:
		return "Removal"
	case RoleDamage:
		return "Damage"
	default:
		return "None"
	}
}

// ClassifyCombat returns the combat role of a card. RoleNone means the card
// is not a combat trick worth holding. Sorcery-speed cards always return
// RoleNone — the holding logic only makes sense for instants.
//
// Bias: the classifier prefers false negatives (RoleNone) over false
// positives. An unrecognised pump degrades to today's behaviour (cast
// pre-combat); a misclassified sorcery would cause the AI to sit on a card
// it should cast.
func ClassifyCombat(card mage.Card) CombatRole {
	if card == nil || !card.HasType(core.TypeInstant) {
		return RoleNone
	}

	// Walk every spell ability's effects and pick the strongest signal.
	// Pump > Damage > Removal > Protection in priority — most cards have one
	// dominant effect, but composite cards (e.g., "creature gets +2/+0 and
	// deals 2 damage") should be treated as Pump because the buff is what
	// drives the combat decision.
	role := RoleNone
	for _, ab := range card.Abilities() {
		def, ok := ab.(*mage.ActionDefinition)
		if !ok || def.Kind() != mage.ActionSpell {
			continue
		}
		for _, e := range def.Effects() {
			r := classifyEffect(e)
			if rolePriority(r) > rolePriority(role) {
				role = r
			}
		}
	}
	return role
}

// ClassifyAbility returns the combat role of an activated ability on a
// permanent. Same priority logic as ClassifyCombat.
//
// Note: many activated abilities require {T} or have one-shot constraints
// (sacrifice, "once per turn") — those are still combat-eligible so long as
// the ability is currently activatable. Caller is expected to enumerate via
// g.GetActivatableAbilities and only invoke ClassifyAbility on the result.
func ClassifyAbility(def *mage.ActionDefinition) CombatRole {
	if def == nil || def.Kind() != mage.ActionActivated {
		return RoleNone
	}
	role := RoleNone
	for _, e := range def.Effects() {
		r := classifyEffect(e)
		if rolePriority(r) > rolePriority(role) {
			role = r
		}
	}
	return role
}

func classifyEffect(e mage.Effect) CombatRole {
	if e == nil {
		return RoleNone
	}
	props := e.Properties()
	if props.PowerBoost > 0 || props.ToughnessBoost > 0 {
		return RolePump
	}
	if props.DamageValue != nil {
		return RoleDamage
	}
	if props.Outcome == mage.OutcomeDetriment && !props.Mass {
		// Single-target detriment on a creature target = removal.
		// Mass effects (wraths) aren't combat tricks in the hold-for-window sense.
		return RoleRemoval
	}
	return RoleNone
}

// rolePriority orders roles for "strongest signal wins" classification. Pump
// dominates because it's the most context-sensitive (target's P/T determines
// whether to cast pre- or post-block); damage and removal are reactive too
// but are slightly less likely to be miscast pre-combat.
func rolePriority(r CombatRole) int {
	switch r {
	case RolePump:
		return 4
	case RoleDamage:
		return 3
	case RoleRemoval:
		return 2
	case RoleProtection:
		return 1
	default:
		return 0
	}
}
