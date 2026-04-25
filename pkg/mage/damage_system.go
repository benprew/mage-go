package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// DamageSystem manages damage-related state. Most replacement-based damage
// mechanisms have been migrated to the ReplacementEffect system. What remains
// here is damage reflection (post-damage, not a replacement) and the legacy
// AddDamagePreventionRule API which delegates to the EffectManager's replacement list.
type DamageSystem struct {
	damageReflection map[uuid.UUID]damageReflectionEntry // player -> Eye for an Eye reflection info (one-shot)
	em               *EffectManager                      // back-reference for AddDamagePreventionRule delegation
}

// damageReflectionEntry tracks the Eye for an Eye reflection state for a player.
type damageReflectionEntry struct {
	eyeSourceID  uuid.UUID // the Eye for an Eye card (used as damage source attribution)
	chosenSource uuid.UUID // the source the player chose (uuid.Nil = any source)
}

// NewDamageSystem creates a DamageSystem with all maps initialized.
func NewDamageSystem() *DamageSystem {
	return &DamageSystem{
		damageReflection: make(map[uuid.UUID]damageReflectionEntry),
	}
}

// SetEffectManager sets the back-reference to the EffectManager.
func (ds *DamageSystem) SetEffectManager(em *EffectManager) {
	ds.em = em
}

// ResetPerCycle resets all state that is recomputed each Apply() cycle.
func (ds *DamageSystem) ResetPerCycle() {
	// Nothing to reset — all per-cycle state is now on the replacement system
}

// ClearEndOfTurn resets all turn-scoped damage state.
func (ds *DamageSystem) ClearEndOfTurn() {
	ds.damageReflection = make(map[uuid.UUID]damageReflectionEntry)
}

// ---------------------------------------------------------------------------
// Damage Reflection (Eye for an Eye) — stays here as a post-damage effect
// ---------------------------------------------------------------------------

// SetDamageReflection sets a one-shot damage reflection for a player (Eye for an Eye).
func (ds *DamageSystem) SetDamageReflection(playerID, eyeSourceID, chosenSourceID uuid.UUID) {
	ds.damageReflection[playerID] = damageReflectionEntry{
		eyeSourceID:  eyeSourceID,
		chosenSource: chosenSourceID,
	}
}

// GetDamageReflection returns the reflection entry if reflection is active for the player.
func (ds *DamageSystem) GetDamageReflection(playerID uuid.UUID) (damageReflectionEntry, bool) {
	entry, ok := ds.damageReflection[playerID]
	return entry, ok
}

// ClearDamageReflection clears the damage reflection for a player.
func (ds *DamageSystem) ClearDamageReflection(playerID uuid.UUID) {
	delete(ds.damageReflection, playerID)
}

// ---------------------------------------------------------------------------
// Legacy API: AddDamagePreventionRule — delegates to the replacement system
// ---------------------------------------------------------------------------

// damagePreventionRule is the old internal struct used by the options API.
type damagePreventionRule struct {
	from       PermanentFilter
	to         PermanentFilter
	oneShot    bool
	combatOnly bool
	playerOnly bool
}

type damagePreventionRuleOption func(*damagePreventionRule)

// WithFrom sets the "from" filter for a damage prevention rule.
func WithFrom(from PermanentFilter) damagePreventionRuleOption {
	return func(dpr *damagePreventionRule) {
		dpr.from = from
	}
}

// WithTo sets the "to" filter for a damage prevention rule.
func WithTo(to PermanentFilter) damagePreventionRuleOption {
	return func(dpr *damagePreventionRule) {
		dpr.to = to
	}
}

// WithOneShot marks a rule as consumed after first match.
func WithOneShot(oneshot bool) damagePreventionRuleOption {
	return func(dpr *damagePreventionRule) {
		dpr.oneShot = oneshot
	}
}

// WithCombatOnly restricts a damage prevention rule to combat damage only.
func WithCombatOnly() damagePreventionRuleOption {
	return func(dpr *damagePreventionRule) {
		dpr.combatOnly = true
	}
}

// WithPlayerOnly restricts a damage prevention rule to damage dealt to players only
// (ignores damage dealt to permanents).
func WithPlayerOnly() damagePreventionRuleOption {
	return func(dpr *damagePreventionRule) {
		dpr.playerOnly = true
	}
}

// AddDamagePreventionRule registers a damage prevention rule by delegating to the
// EffectManager's replacement system. This preserves the existing card API.
func (ds *DamageSystem) AddDamagePreventionRule(opts ...damagePreventionRuleOption) {
	dpr := &damagePreventionRule{}
	for _, opt := range opts {
		opt(dpr)
	}
	if ds.em != nil {
		ds.em.AddCycleReplacement(&damagePreventionRuleReplacement{
			from:       dpr.from,
			to:         dpr.to,
			oneShot:    dpr.oneShot,
			combatOnly: dpr.combatOnly,
			playerOnly: dpr.playerOnly,
		})
	}
}

// AddRegenerationShield is a legacy API that delegates to the replacement system.
func (ds *DamageSystem) AddRegenerationShield(targetID uuid.UUID) {
	if ds.em == nil {
		return
	}
	for _, r := range ds.em.replacements {
		if regen, ok := r.(*regenerationReplacement); ok && regen.permanentID == targetID {
			regen.shields++
			return
		}
	}
	ds.em.AddReplacement(&regenerationReplacement{replacementBase: replacementBase{duration: EndOfTurn}, permanentID: targetID, shields: 1})
}

// SetArtifactDamageRedirect is a legacy API that delegates to the replacement system.
func (ds *DamageSystem) SetArtifactDamageRedirect(controllerID, permID uuid.UUID) {
	if ds.em != nil {
		ds.em.AddCycleReplacement(&artifactDamageRedirectReplacement{
			controllerID:   controllerID,
			redirectPermID: permID,
		})
	}
}
