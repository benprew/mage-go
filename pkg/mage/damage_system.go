package mage

import (
	. "github.com/mage/mage/pkg/mage/core"
	"github.com/google/uuid"
)

// DamageSystem manages all damage-related state: prevention shields,
// regeneration, damage redirection, reflection, and combat damage prevention.
// It is owned by EffectManager and exposed as a public field.
type DamageSystem struct {
	regenerationShields    map[uuid.UUID]int
	preventionShields      map[uuid.UUID]int
	preventionRules        []damagePreventionRule
	colorPrevention        map[uuid.UUID][]Color    // player -> colors that prevent next damage source
	typePrevention         map[uuid.UUID][]CardType // player -> card types that prevent next damage source
	reverseDamageShields   map[uuid.UUID]bool       // players with Reverse Damage active this turn
	forcefieldShields      map[uuid.UUID]bool       // players with Forcefield active this turn
	preventCombatDamage    bool                     // true if all combat damage is prevented this turn (Fog, etc.)
	bodyguard              map[uuid.UUID]uuid.UUID  // controller -> bodyguard permanent ID (Veteran Bodyguard)
	playerDamageRedirect   map[uuid.UUID]uuid.UUID  // controller -> creature that absorbs ALL damage to player
	artifactDamageRedirect map[uuid.UUID]uuid.UUID  // controller -> creature that absorbs artifact damage to player (Martyrs of Korlis)
	creatureDamageRedirect map[uuid.UUID]uuid.UUID  // creature -> player who receives damage instead of creature (one-shot)
	damageReflection       map[uuid.UUID]damageReflectionEntry // player -> Eye for an Eye reflection info (one-shot)
	drawReplacement        map[uuid.UUID]int                  // player -> X value for Aladdin's Lamp draw replacement
}

// damageReflectionEntry tracks the Eye for an Eye reflection state for a player.
type damageReflectionEntry struct {
	eyeSourceID  uuid.UUID // the Eye for an Eye card (used as damage source attribution)
	chosenSource uuid.UUID // the source the player chose (uuid.Nil = any source)
}

// NewDamageSystem creates a DamageSystem with all maps initialized.
func NewDamageSystem() *DamageSystem {
	return &DamageSystem{
		regenerationShields:    make(map[uuid.UUID]int),
		preventionShields:      make(map[uuid.UUID]int),
		forcefieldShields:      make(map[uuid.UUID]bool),
		reverseDamageShields:   make(map[uuid.UUID]bool),
		colorPrevention:        make(map[uuid.UUID][]Color),
		typePrevention:         make(map[uuid.UUID][]CardType),
		bodyguard:              make(map[uuid.UUID]uuid.UUID),
		playerDamageRedirect:   make(map[uuid.UUID]uuid.UUID),
		artifactDamageRedirect: make(map[uuid.UUID]uuid.UUID),
		creatureDamageRedirect: make(map[uuid.UUID]uuid.UUID),
		damageReflection:       make(map[uuid.UUID]damageReflectionEntry),
		drawReplacement:        make(map[uuid.UUID]int),
	}
}

// ResetPerCycle resets all state that is recomputed each Apply() cycle.
func (ds *DamageSystem) ResetPerCycle() {
	ds.bodyguard = make(map[uuid.UUID]uuid.UUID)
	ds.playerDamageRedirect = make(map[uuid.UUID]uuid.UUID)
	ds.artifactDamageRedirect = make(map[uuid.UUID]uuid.UUID)
	// Rebuild prevention rules from continuous effects; preserve one-shot rules (e.g. CoP)
	var oneShotRules []damagePreventionRule
	for _, r := range ds.preventionRules {
		if r.oneShot {
			oneShotRules = append(oneShotRules, r)
		}
	}
	ds.preventionRules = oneShotRules
}

// ClearEndOfTurn resets all turn-scoped damage state.
func (ds *DamageSystem) ClearEndOfTurn() {
	ds.preventCombatDamage = false
	ds.preventionShields = make(map[uuid.UUID]int)
	ds.preventionRules = nil
	ds.forcefieldShields = make(map[uuid.UUID]bool)
	ds.damageReflection = make(map[uuid.UUID]damageReflectionEntry)
	ds.drawReplacement = make(map[uuid.UUID]int)
}

// ---------------------------------------------------------------------------
// Forcefield
// ---------------------------------------------------------------------------

// AddForcefieldShield marks a player as having Forcefield active this turn.
func (ds *DamageSystem) AddForcefieldShield(playerID uuid.UUID) {
	ds.forcefieldShields[playerID] = true
}

// HasForcefieldShield returns true if the player has Forcefield active this turn.
func (ds *DamageSystem) HasForcefieldShield(playerID uuid.UUID) bool {
	return ds.forcefieldShields[playerID]
}

// ClearForcefieldShields resets Forcefield shields at end of turn.
func (ds *DamageSystem) ClearForcefieldShields() {
	ds.forcefieldShields = make(map[uuid.UUID]bool)
}

// ---------------------------------------------------------------------------
// Bodyguard / Damage Redirection
// ---------------------------------------------------------------------------

// SetBodyguard marks a bodyguard permanent for a player (Veteran Bodyguard).
func (ds *DamageSystem) SetBodyguard(controllerID, permID uuid.UUID) {
	ds.bodyguard[controllerID] = permID
}

// GetBodyguard returns the bodyguard permanent ID for a player, or uuid.Nil if none.
func (ds *DamageSystem) GetBodyguard(controllerID uuid.UUID) uuid.UUID {
	return ds.bodyguard[controllerID]
}

// ClearBodyguard clears the bodyguard for a player.
func (ds *DamageSystem) ClearBodyguard(controllerID uuid.UUID) {
	delete(ds.bodyguard, controllerID)
}

// SetPlayerDamageRedirect sets a creature that absorbs ALL damage dealt to a player.
func (ds *DamageSystem) SetPlayerDamageRedirect(controllerID, permID uuid.UUID) {
	ds.playerDamageRedirect[controllerID] = permID
}

// GetPlayerDamageRedirect returns the creature absorbing damage for a player, or uuid.Nil.
func (ds *DamageSystem) GetPlayerDamageRedirect(controllerID uuid.UUID) uuid.UUID {
	return ds.playerDamageRedirect[controllerID]
}

// SetArtifactDamageRedirect sets a creature that absorbs artifact damage dealt to a player.
func (ds *DamageSystem) SetArtifactDamageRedirect(controllerID, permID uuid.UUID) {
	ds.artifactDamageRedirect[controllerID] = permID
}

// GetArtifactDamageRedirect returns the creature absorbing artifact damage for a player, or uuid.Nil.
func (ds *DamageSystem) GetArtifactDamageRedirect(controllerID uuid.UUID) uuid.UUID {
	return ds.artifactDamageRedirect[controllerID]
}

// SetCreatureDamageRedirect sets a one-shot redirect: next damage dealt to creatureID
// is dealt to targetPlayerID instead (Jade Monolith).
func (ds *DamageSystem) SetCreatureDamageRedirect(creatureID, targetPlayerID uuid.UUID) {
	ds.creatureDamageRedirect[creatureID] = targetPlayerID
}

// GetCreatureDamageRedirect returns the player who should receive damage instead of a creature, or uuid.Nil.
func (ds *DamageSystem) GetCreatureDamageRedirect(creatureID uuid.UUID) uuid.UUID {
	return ds.creatureDamageRedirect[creatureID]
}

// ClearCreatureDamageRedirect clears the one-shot creature damage redirect.
func (ds *DamageSystem) ClearCreatureDamageRedirect(creatureID uuid.UUID) {
	delete(ds.creatureDamageRedirect, creatureID)
}

// ---------------------------------------------------------------------------
// Damage Reflection (Eye for an Eye)
// ---------------------------------------------------------------------------

// SetDamageReflection sets a one-shot damage reflection for a player (Eye for an Eye).
// eyeSourceID is the Eye for an Eye card's ID (used as damage source attribution).
// chosenSourceID is the specific source the player chose (uuid.Nil = any source).
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

// ClearAllDamageReflections resets all damage reflections at end of turn.
func (ds *DamageSystem) ClearAllDamageReflections() {
	ds.damageReflection = make(map[uuid.UUID]damageReflectionEntry)
}

// ---------------------------------------------------------------------------
// Draw Replacement (Aladdin's Lamp)
// ---------------------------------------------------------------------------

// SetDrawReplacement stores a pending draw replacement for a player (Aladdin's Lamp).
// count is the number of cards to look at (X value).
func (ds *DamageSystem) SetDrawReplacement(playerID uuid.UUID, count int) {
	ds.drawReplacement[playerID] = count
}

// GetDrawReplacement returns the pending draw replacement count, or 0 if none.
func (ds *DamageSystem) GetDrawReplacement(playerID uuid.UUID) (int, bool) {
	count, ok := ds.drawReplacement[playerID]
	return count, ok
}

// ClearDrawReplacement clears the pending draw replacement for a player.
func (ds *DamageSystem) ClearDrawReplacement(playerID uuid.UUID) {
	delete(ds.drawReplacement, playerID)
}

// ClearAllDrawReplacements resets all draw replacements at end of turn.
func (ds *DamageSystem) ClearAllDrawReplacements() {
	ds.drawReplacement = make(map[uuid.UUID]int)
}

// ---------------------------------------------------------------------------
// Regeneration
// ---------------------------------------------------------------------------

// AddRegenerationShield increments the regeneration shield count for a permanent.
func (ds *DamageSystem) AddRegenerationShield(targetID uuid.UUID) {
	ds.regenerationShields[targetID]++
}

// ConsumeRegenerationShield returns true and decrements if a shield is available.
func (ds *DamageSystem) ConsumeRegenerationShield(targetID uuid.UUID) bool {
	if ds.regenerationShields[targetID] > 0 {
		ds.regenerationShields[targetID]--
		return true
	}
	return false
}

// ClearRegenerationShields clears all regeneration shields for permanents
// controlled by the given player (per MTG rules, cleared during untap step).
func (ds *DamageSystem) ClearRegenerationShields(playerID uuid.UUID, g *Game) {
	for _, p := range g.Battlefield {
		if p.Controller == playerID {
			delete(ds.regenerationShields, p.ID())
		}
	}
}

// ---------------------------------------------------------------------------
// Prevention Shields
// ---------------------------------------------------------------------------

// AddPreventionShield adds to the damage prevention shield for a permanent.
func (ds *DamageSystem) AddPreventionShield(targetID uuid.UUID, amount int) {
	ds.preventionShields[targetID] += amount
}

// PreventDamage consumes prevention shields to prevent damage, returning the
// amount actually prevented.
func (ds *DamageSystem) PreventDamage(targetID uuid.UUID, amount int) int {
	shield := ds.preventionShields[targetID]
	if shield <= 0 {
		return 0
	}
	prevented := min(amount, shield)
	ds.preventionShields[targetID] -= prevented
	return prevented
}

// ClearPreventionShields resets all damage prevention shields.
func (ds *DamageSystem) ClearPreventionShields() {
	ds.preventionShields = make(map[uuid.UUID]int)
}

// ---------------------------------------------------------------------------
// Reverse Damage
// ---------------------------------------------------------------------------

// AddReverseDamageShield marks a player as having Reverse Damage active.
func (ds *DamageSystem) AddReverseDamageShield(playerID uuid.UUID) {
	ds.reverseDamageShields[playerID] = true
}

// HasReverseDamageShield returns true if the player has Reverse Damage active.
func (ds *DamageSystem) HasReverseDamageShield(playerID uuid.UUID) bool {
	return ds.reverseDamageShields[playerID]
}

// ClearReverseDamageShield clears Reverse Damage shield for a player.
func (ds *DamageSystem) ClearReverseDamageShield(playerID uuid.UUID) {
	delete(ds.reverseDamageShields, playerID)
}

// ---------------------------------------------------------------------------
// Combat Damage Prevention (Fog)
// ---------------------------------------------------------------------------

// SetPreventCombatDamage marks all combat damage as prevented this turn (Fog, etc.).
func (ds *DamageSystem) SetPreventCombatDamage() {
	ds.preventCombatDamage = true
}

// PreventsCombatDamage returns true if all combat damage is prevented this turn.
func (ds *DamageSystem) PreventsCombatDamage() bool {
	return ds.preventCombatDamage
}

// ClearPreventCombatDamage resets combat damage prevention at end of turn.
func (ds *DamageSystem) ClearPreventCombatDamage() {
	ds.preventCombatDamage = false
}

// ---------------------------------------------------------------------------
// Color / Type Prevention
// ---------------------------------------------------------------------------

// AddTypePrevention adds a card-type prevention shield (prevents all damage from one source of that type).
func (ds *DamageSystem) AddTypePrevention(playerID uuid.UUID, ct CardType) {
	ds.typePrevention[playerID] = append(ds.typePrevention[playerID], ct)
}

// CheckTypePrevention returns true and consumes a shield if the player has
// type prevention matching the source's card type.
func (ds *DamageSystem) CheckTypePrevention(playerID uuid.UUID, sourceCard Card) bool {
	types := ds.typePrevention[playerID]
	if len(types) == 0 || sourceCard == nil {
		return false
	}
	for i, shield := range types {
		if sourceCard.HasType(shield) {
			// Consume this shield
			ds.typePrevention[playerID] = append(types[:i], types[i+1:]...)
			return true
		}
	}
	return false
}

// AddColorPrevention adds a color prevention shield (prevents all damage from one source of that color).
func (ds *DamageSystem) AddColorPrevention(playerID uuid.UUID, color Color) {
	ds.colorPrevention[playerID] = append(ds.colorPrevention[playerID], color)
}

// CheckColorPrevention returns true and consumes a shield if the player has
// color prevention matching the source's color.
func (ds *DamageSystem) CheckColorPrevention(playerID uuid.UUID, sourceCard Card) bool {
	colors := ds.colorPrevention[playerID]
	if len(colors) == 0 || sourceCard == nil {
		return false
	}
	sourceColors := sourceCard.ManaCost().Colors()
	for i, shield := range colors {
		for _, sc := range sourceColors {
			if sc == shield {
				// Consume this shield
				ds.colorPrevention[playerID] = append(colors[:i], colors[i+1:]...)
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Damage Prevention Rules
// ---------------------------------------------------------------------------

type damagePreventionRule struct {
	from    PermanentFilter
	to      PermanentFilter
	oneShot bool
}

type damagePreventionRuleOption func(*damagePreventionRule)

func WithFrom(from PermanentFilter) damagePreventionRuleOption {
	return func(dpr *damagePreventionRule) {
		dpr.from = from
	}
}

func WithTo(to PermanentFilter) damagePreventionRuleOption {
	return func(dpr *damagePreventionRule) {
		dpr.to = to
	}
}

func WithOneShot(oneshot bool) damagePreventionRuleOption {
	return func(dpr *damagePreventionRule) {
		dpr.oneShot = oneshot
	}
}

// AddDamagePreventionRule registers a damage prevention rule.
func (ds *DamageSystem) AddDamagePreventionRule(opts ...damagePreventionRuleOption) {
	dpr := &damagePreventionRule{}
	for _, opt := range opts {
		opt(dpr)
	}
	ds.preventionRules = append(ds.preventionRules, *dpr)
}

// ClearDamagePreventionRules clears all damage prevention rules.
func (ds *DamageSystem) ClearDamagePreventionRules() {
	ds.preventionRules = make([]damagePreventionRule, 0)
}

// CheckDamagePreventionRules returns true if any rule matches the given
// source and target permanents, meaning all damage should be prevented.
// One-shot rules are consumed on use.
func (ds *DamageSystem) CheckDamagePreventionRules(source, target *Permanent, g *Game) bool {
	for i, rule := range ds.preventionRules {
		fromMatch := rule.from.IsZero() || (source != nil && rule.from.Match(source, g))
		toMatch := rule.to.Match(target, g)
		if fromMatch && toMatch {
			if rule.oneShot {
				ds.preventionRules = append(ds.preventionRules[:i], ds.preventionRules[i+1:]...)
			}
			return true
		}
	}
	return false
}
