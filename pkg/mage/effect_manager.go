package mage

import (
	. "github.com/mage/mage/pkg/mage/core"
	"github.com/google/uuid"
)

// ContinuousEffect represents an ongoing effect on the game.
type ContinuousEffect interface {
	Apply(g *Game) error
	GetLayer() Layer
	GetDuration() Duration
	IsActive(g *Game) bool
	SourceID() uuid.UUID
	SetSourceID(uuid.UUID)
}

// effectSource provides SourceID/SetSourceID to all continuous effects.
type effectSource struct {
	sourceID uuid.UUID
}

func (s *effectSource) SourceID() uuid.UUID      { return s.sourceID }
func (s *effectSource) SetSourceID(id uuid.UUID) { s.sourceID = id }

// ---------------------------------------------------------------------------
// Compositional continuous effect primitives
// ---------------------------------------------------------------------------

// ActiveCondition determines when a continuous effect is active.
type ActiveCondition func(g *Game, sourceID uuid.UUID) bool

// SourceOnBattlefield is active while the source permanent exists on the battlefield.
// This is the default for FuncContinuousEffect so you rarely need to pass it explicitly.
var SourceOnBattlefield ActiveCondition = func(g *Game, sourceID uuid.UUID) bool {
	return g.FindPermanent(sourceID) != nil
}

// SourceAttached is active while the source is on the battlefield and attached to another permanent.
var SourceAttached ActiveCondition = func(g *Game, sourceID uuid.UUID) bool {
	src := g.FindPermanent(sourceID)
	return src != nil && src.IsAttached()
}

// SourceUntapped is active while the source is on the battlefield and untapped.
var SourceUntapped ActiveCondition = func(g *Game, sourceID uuid.UUID) bool {
	src := g.FindPermanent(sourceID)
	return src != nil && !src.Tapped
}

// TargetOnBattlefield returns an ActiveCondition that's active while a specific permanent exists.
func TargetOnBattlefield(targetID uuid.UUID) ActiveCondition {
	return func(g *Game, _ uuid.UUID) bool {
		return g.FindPermanent(targetID) != nil
	}
}

// WithSourceCondition bridges the existing SourceCondition type to ActiveCondition.
func WithSourceCondition(cond SourceCondition) ActiveCondition {
	return func(g *Game, sourceID uuid.UUID) bool {
		src := g.FindPermanent(sourceID)
		if src == nil {
			return false
		}
		return cond(src, g)
	}
}

// ContinuousApplyFunc is the apply function for a functional continuous effect.
type ContinuousApplyFunc func(g *Game, sourceID uuid.UUID) error

// funcContinuousEffect implements ContinuousEffect using a function and metadata.
type funcContinuousEffect struct {
	layer    Layer
	duration Duration
	apply    ContinuousApplyFunc
	active   ActiveCondition // nil means SourceOnBattlefield
	effectSource
}

// FuncContinuousEffect creates a ContinuousEffect from a layer, duration, and apply function.
// By default, it's active while the source permanent is on the battlefield.
// Pass an optional ActiveCondition to customize when it's active.
func FuncContinuousEffect(layer Layer, duration Duration, apply ContinuousApplyFunc, condition ...ActiveCondition) ContinuousEffect {
	var cond ActiveCondition
	if len(condition) > 0 {
		cond = condition[0]
	}
	return &funcContinuousEffect{
		layer:    layer,
		duration: duration,
		apply:    apply,
		active:   cond,
	}
}

func (e *funcContinuousEffect) GetLayer() Layer       { return e.layer }
func (e *funcContinuousEffect) GetDuration() Duration { return e.duration }

func (e *funcContinuousEffect) IsActive(g *Game) bool {
	if e.active != nil {
		return e.active(g, e.sourceID)
	}
	return g.FindPermanent(e.sourceID) != nil
}

func (e *funcContinuousEffect) Apply(g *Game) error {
	return e.apply(g, e.sourceID)
}

// AttachedApplyFunc receives both the source and the attached target.
type AttachedApplyFunc func(g *Game, source, target *Permanent) error

// attachedEffect implements ContinuousEffect for aura/equipment patterns.
type attachedEffect struct {
	layer Layer
	apply AttachedApplyFunc
	effectSource
}

// AttachedEffect creates a ContinuousEffect that applies while the source is
// attached to another permanent. The apply function receives both source and target.
func AttachedEffect(layer Layer, apply AttachedApplyFunc) ContinuousEffect {
	return &attachedEffect{
		layer: layer,
		apply: apply,
	}
}

func (e *attachedEffect) GetLayer() Layer       { return e.layer }
func (e *attachedEffect) GetDuration() Duration { return WhileOnBattlefield }

func (e *attachedEffect) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	return src != nil && src.IsAttached()
}

func (e *attachedEffect) Apply(g *Game) error {
	src := g.FindPermanent(e.sourceID)
	if src == nil || !src.IsAttached() {
		return nil
	}
	target := g.FindPermanent(src.AttachedTo)
	if target == nil {
		return nil
	}
	return e.apply(g, src, target)
}

// TargetApplyFunc receives the targeted permanent.
type TargetApplyFunc func(g *Game, target *Permanent) error

// targetEffect implements ContinuousEffect for effects targeting a specific permanent.
type targetEffect struct {
	layer    Layer
	duration Duration
	targetID uuid.UUID
	apply    TargetApplyFunc
	effectSource
}

// TargetEffect creates a ContinuousEffect that applies to a specific permanent by ID.
// Active while the target permanent exists on the battlefield.
func TargetEffect(layer Layer, duration Duration, targetID uuid.UUID, apply TargetApplyFunc) ContinuousEffect {
	return &targetEffect{
		layer:    layer,
		duration: duration,
		targetID: targetID,
		apply:    apply,
	}
}

func (e *targetEffect) GetLayer() Layer       { return e.layer }
func (e *targetEffect) GetDuration() Duration { return e.duration }

func (e *targetEffect) IsActive(g *Game) bool {
	return g.FindPermanent(e.targetID) != nil
}

func (e *targetEffect) Apply(g *Game) error {
	target := g.FindPermanent(e.targetID)
	if target == nil {
		return nil
	}
	return e.apply(g, target)
}

// EffectManager manages and applies continuous effects.
type EffectManager struct {
	effects                []ContinuousEffect
	powerBonuses           map[uuid.UUID]int
	toughBonuses           map[uuid.UUID]int
	attrDeltas             map[uuid.UUID]map[Attr]int // deltas accumulated during Apply(); written to perm.grantedAttrs
	regenerationShields    map[uuid.UUID]int
	preventionShields      map[uuid.UUID]int
	preventionRules        []damagePreventionRule
	landUntapLimit         int                     // -1 = no limit; >= 0 = max lands that may untap per turn
	forcefieldShields      map[uuid.UUID]bool      // players with Forcefield active this turn
	reverseDamageShields   map[uuid.UUID]bool      // players with Reverse Damage active this turn
	channelActive          map[uuid.UUID]bool      // players with Channel active this turn
	colorPrevention        map[uuid.UUID][]Color   // player -> colors that prevent next damage source
	spellCostIncrease      map[Color]int           // color -> additional generic cost for spells of that color
	unlimitedLandPlays     bool                    // true if a player can play unlimited lands (Fastbond)
	sanctuaryActive        map[uuid.UUID]bool      // player -> if true, only flying/islandwalk can attack them
	lichActive             map[uuid.UUID]uuid.UUID // player -> source permanent ID of active Lich
	skipNextDraw           map[uuid.UUID]bool      // player -> if true, skip normal draw in draw step
	manaConversion         map[Color]Color         // from color -> to color (Sunglasses of Urza)
	bodyguard              map[uuid.UUID]uuid.UUID // controller -> bodyguard permanent ID (Veteran Bodyguard)
	playerDamageRedirect   map[uuid.UUID]uuid.UUID // controller -> creature that absorbs ALL damage to player
	creatureDamageRedirect map[uuid.UUID]uuid.UUID // creature -> player who receives damage instead of creature (one-shot)
	preventCombatDamage    bool                    // true if all combat damage is prevented this turn (Fog, etc.)
}

func NewEffectManager() *EffectManager {
	return &EffectManager{
		powerBonuses:           make(map[uuid.UUID]int),
		toughBonuses:           make(map[uuid.UUID]int),
		attrDeltas:             make(map[uuid.UUID]map[Attr]int),
		regenerationShields:    make(map[uuid.UUID]int),
		preventionShields:      make(map[uuid.UUID]int),
		landUntapLimit:         -1,
		forcefieldShields:      make(map[uuid.UUID]bool),

		reverseDamageShields:   make(map[uuid.UUID]bool),
		channelActive:          make(map[uuid.UUID]bool),
		colorPrevention:        make(map[uuid.UUID][]Color),
		spellCostIncrease:      make(map[Color]int),
		sanctuaryActive:        make(map[uuid.UUID]bool),
		lichActive:             make(map[uuid.UUID]uuid.UUID),
		skipNextDraw:           make(map[uuid.UUID]bool),
		manaConversion:         make(map[Color]Color),
		bodyguard:              make(map[uuid.UUID]uuid.UUID),
		playerDamageRedirect:   make(map[uuid.UUID]uuid.UUID),
		creatureDamageRedirect: make(map[uuid.UUID]uuid.UUID),
	}
}

// AddForcefieldShield marks a player as having Forcefield active this turn.
func (em *EffectManager) AddForcefieldShield(playerID uuid.UUID) {
	em.forcefieldShields[playerID] = true
}

// HasForcefieldShield returns true if the player has Forcefield active this turn.
func (em *EffectManager) HasForcefieldShield(playerID uuid.UUID) bool {
	return em.forcefieldShields[playerID]
}

// ClearForcefieldShields resets Forcefield shields at end of turn.
func (em *EffectManager) ClearForcefieldShields() {
	em.forcefieldShields = make(map[uuid.UUID]bool)
}

// LandUntapLimit returns the current land untap limit. -1 means no limit.
func (em *EffectManager) LandUntapLimit() int {
	return em.landUntapLimit
}

// HasUnlimitedLandPlays returns true if a player can play unlimited lands this turn.
func (em *EffectManager) HasUnlimitedLandPlays() bool {
	return em.unlimitedLandPlays
}

// SetSanctuaryActive marks a player as protected by Island Sanctuary.
func (em *EffectManager) SetSanctuaryActive(playerID uuid.UUID) {
	em.sanctuaryActive[playerID] = true
}

// IsSanctuaryActive returns true if the player is protected by Island Sanctuary.
func (em *EffectManager) IsSanctuaryActive(playerID uuid.UUID) bool {
	return em.sanctuaryActive[playerID]
}

// ClearSanctuary clears Island Sanctuary protection for a player.
func (em *EffectManager) ClearSanctuary(playerID uuid.UUID) {
	delete(em.sanctuaryActive, playerID)
}

// SetLichActive records the source permanent ID of a Lich controlled by playerID.
func (em *EffectManager) SetLichActive(playerID, sourceID uuid.UUID) {
	em.lichActive[playerID] = sourceID
}

// IsLichActive returns true if the player has an active Lich still on the battlefield.
func (em *EffectManager) IsLichActive(g *Game, playerID uuid.UUID) bool {
	sourceID, ok := em.lichActive[playerID]
	if !ok {
		return false
	}
	return g.FindPermanent(sourceID) != nil
}

// ClearLich clears Lich replacement effects for a player.
func (em *EffectManager) ClearLich(playerID uuid.UUID) {
	delete(em.lichActive, playerID)
}

// SetSkipNextDraw marks a player to skip their next draw step draw.
func (em *EffectManager) SetSkipNextDraw(playerID uuid.UUID) {
	em.skipNextDraw[playerID] = true
}

// ShouldSkipDraw returns true and clears the flag if the player should skip their draw.
func (em *EffectManager) ShouldSkipDraw(playerID uuid.UUID) bool {
	if em.skipNextDraw[playerID] {
		delete(em.skipNextDraw, playerID)
		return true
	}
	return false
}


// SetManaConversion sets a mana color conversion (e.g. Red→White for Sunglasses of Urza).
func (em *EffectManager) SetManaConversion(from, to Color) {
	em.manaConversion[from] = to
}

// GetManaConversion returns the converted color for a given color, if any.
func (em *EffectManager) GetManaConversion(from Color) (Color, bool) {
	to, ok := em.manaConversion[from]
	return to, ok
}

// ClearManaConversion clears all mana conversions.
func (em *EffectManager) ClearManaConversion() {
	em.manaConversion = make(map[Color]Color)
}

// SetBodyguard marks a bodyguard permanent for a player (Veteran Bodyguard).
func (em *EffectManager) SetBodyguard(controllerID, permID uuid.UUID) {
	em.bodyguard[controllerID] = permID
}

// GetBodyguard returns the bodyguard permanent ID for a player, or uuid.Nil if none.
func (em *EffectManager) GetBodyguard(controllerID uuid.UUID) uuid.UUID {
	return em.bodyguard[controllerID]
}

// ClearBodyguard clears the bodyguard for a player.
func (em *EffectManager) ClearBodyguard(controllerID uuid.UUID) {
	delete(em.bodyguard, controllerID)
}

// SetPlayerDamageRedirect sets a creature that absorbs ALL damage dealt to a player.
func (em *EffectManager) SetPlayerDamageRedirect(controllerID, permID uuid.UUID) {
	em.playerDamageRedirect[controllerID] = permID
}

// GetPlayerDamageRedirect returns the creature absorbing damage for a player, or uuid.Nil.
func (em *EffectManager) GetPlayerDamageRedirect(controllerID uuid.UUID) uuid.UUID {
	return em.playerDamageRedirect[controllerID]
}

// SetCreatureDamageRedirect sets a one-shot redirect: next damage dealt to creatureID
// is dealt to targetPlayerID instead (Jade Monolith).
func (em *EffectManager) SetCreatureDamageRedirect(creatureID, targetPlayerID uuid.UUID) {
	em.creatureDamageRedirect[creatureID] = targetPlayerID
}

// GetCreatureDamageRedirect returns the player who should receive damage instead of a creature, or uuid.Nil.
func (em *EffectManager) GetCreatureDamageRedirect(creatureID uuid.UUID) uuid.UUID {
	return em.creatureDamageRedirect[creatureID]
}

// ClearCreatureDamageRedirect clears the one-shot creature damage redirect.
func (em *EffectManager) ClearCreatureDamageRedirect(creatureID uuid.UUID) {
	delete(em.creatureDamageRedirect, creatureID)
}

// AddRegenerationShield increments the regeneration shield count for a permanent.
func (em *EffectManager) AddRegenerationShield(targetID uuid.UUID) {
	em.regenerationShields[targetID]++
}

// ConsumeRegenerationShield returns true and decrements if a shield is available.
func (em *EffectManager) ConsumeRegenerationShield(targetID uuid.UUID) bool {
	if em.regenerationShields[targetID] > 0 {
		em.regenerationShields[targetID]--
		return true
	}
	return false
}

// ClearRegenerationShields clears all regeneration shields for permanents
// controlled by the given player (per MTG rules, cleared during untap step).
func (em *EffectManager) ClearRegenerationShields(playerID uuid.UUID, g *Game) {
	for _, p := range g.Battlefield {
		if p.Controller == playerID {
			delete(em.regenerationShields, p.ID())
		}
	}
}

// AddPreventionShield adds to the damage prevention shield for a permanent.
func (em *EffectManager) AddPreventionShield(targetID uuid.UUID, amount int) {
	em.preventionShields[targetID] += amount
}

// PreventDamage consumes prevention shields to prevent damage, returning the
// amount actually prevented.
func (em *EffectManager) PreventDamage(targetID uuid.UUID, amount int) int {
	shield := em.preventionShields[targetID]
	if shield <= 0 {
		return 0
	}
	prevented := min(amount, shield)
	em.preventionShields[targetID] -= prevented
	return prevented
}

// ClearPreventionShields resets all damage prevention shields.
func (em *EffectManager) ClearPreventionShields() {
	em.preventionShields = make(map[uuid.UUID]int)
}

// AddReverseDamageShield marks a player as having Reverse Damage active.
func (em *EffectManager) AddReverseDamageShield(playerID uuid.UUID) {
	em.reverseDamageShields[playerID] = true
}

// HasReverseDamageShield returns true if the player has Reverse Damage active.
func (em *EffectManager) HasReverseDamageShield(playerID uuid.UUID) bool {
	return em.reverseDamageShields[playerID]
}

// ClearReverseDamageShield clears Reverse Damage shield for a player.
func (em *EffectManager) ClearReverseDamageShield(playerID uuid.UUID) {
	delete(em.reverseDamageShields, playerID)
}

// SetChannelActive marks a player as having Channel active this turn.
func (em *EffectManager) SetChannelActive(playerID uuid.UUID) {
	em.channelActive[playerID] = true
}

// IsChannelActive returns true if the player has Channel active.
func (em *EffectManager) IsChannelActive(playerID uuid.UUID) bool {
	return em.channelActive[playerID]
}

// ClearChannelActive clears Channel state (called at end of turn).
func (em *EffectManager) ClearChannelActive() {
	em.channelActive = make(map[uuid.UUID]bool)
}

// AddColorPrevention adds a color prevention shield (prevents all damage from one source of that color).
func (em *EffectManager) AddColorPrevention(playerID uuid.UUID, color Color) {
	em.colorPrevention[playerID] = append(em.colorPrevention[playerID], color)
}

// CheckColorPrevention returns true and consumes a shield if the player has
// color prevention matching the source's color.
func (em *EffectManager) CheckColorPrevention(playerID uuid.UUID, sourceCard Card) bool {
	colors := em.colorPrevention[playerID]
	if len(colors) == 0 || sourceCard == nil {
		return false
	}
	sourceColors := sourceCard.ManaCost().Colors()
	for i, shield := range colors {
		for _, sc := range sourceColors {
			if sc == shield {
				// Consume this shield
				em.colorPrevention[playerID] = append(colors[:i], colors[i+1:]...)
				return true
			}
		}
	}
	return false
}

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

func (em *EffectManager) AddDamagePreventionRule(opts ...damagePreventionRuleOption) {
	dpr := &damagePreventionRule{}
	for _, opt := range opts {
		opt(dpr)
	}

	em.preventionRules = append(em.preventionRules, *dpr)
}

func (em *EffectManager) ClearDamagePreventionRules() {
	em.preventionRules = make([]damagePreventionRule, 0)
}

// CheckDamagePreventionRules returns true if any rule matches the given
// source and target permanents, meaning all damage should be prevented.
// One-shot rules are consumed on use.
func (em *EffectManager) CheckDamagePreventionRules(source, target *Permanent, g *Game) bool {
	for i, rule := range em.preventionRules {
		fromMatch := rule.from.IsZero() || (source != nil && rule.from.Match(source, g))
		toMatch := rule.to.Match(target, g)
		if fromMatch && toMatch {
			if rule.oneShot {
				em.preventionRules = append(em.preventionRules[:i], em.preventionRules[i+1:]...)
			}
			return true
		}
	}
	return false
}

// SourceCondition is a predicate checked by preventDamageRuleContinuous to
// decide whether the rule is active. It receives the source permanent and the game.
type SourceCondition func(source *Permanent, g *Game) bool

// WhileSourceAttacking is a SourceCondition that is true only while the source
// permanent is declared as an attacker.
func WhileSourceAttacking(source *Permanent, g *Game) bool {
	return g.Combat != nil && g.Combat.IsAttacking(source.ID())
}

// WhileSourceUntapped is a SourceCondition true only when the source permanent
// is untapped.
func WhileSourceUntapped(source *Permanent, g *Game) bool {
	return !source.Tapped
}

// WhileControlling is a SourceCondition factory, it creates a SourceCondition that
// ensures the source's controller controls a permanent that matches the filter.
func WhileControlling(filter PermanentFilter) SourceCondition {
	return func(source *Permanent, g *Game) bool {
		return g.AnyBattlefield(And(ControlledBy(source.Controller), filter))
	}
}

// preventDamageRuleContinuous registers a damage prevention rule each Apply()
// cycle. The toFactory receives the source permanent's ID so filters can
// reference "self" dynamically.
type preventDamageRuleContinuous struct {
	from      PermanentFilter
	toFactory func(sourceID uuid.UUID) PermanentFilter
	condition SourceCondition // optional; nil means always active while on battlefield
	effectSource
}

// PreventDamageFromTo creates a continuous effect that prevents all damage
// from permanents matching `from` to permanents matching the filter produced
// by `toFactory(sourceID)`. The toFactory pattern lets filters like
// IsBandedWith reference the source permanent's ID. An optional SourceCondition
// controls when the rule is active (e.g. WhileSourceAttacking for Camel).
func PreventDamageFromTo(from PermanentFilter, toFactory func(uuid.UUID) PermanentFilter, condition ...SourceCondition) ContinuousEffect {
	var cond SourceCondition
	if len(condition) > 0 {
		cond = condition[0]
	}
	return &preventDamageRuleContinuous{
		from:      from,
		toFactory: toFactory,
		condition: cond,
	}
}

func (e *preventDamageRuleContinuous) GetLayer() Layer       { return LayerAbility }
func (e *preventDamageRuleContinuous) GetDuration() Duration { return WhileOnBattlefield }

func (e *preventDamageRuleContinuous) IsActive(g *Game) bool {
	src := g.FindPermanent(e.sourceID)
	if src == nil {
		return false
	}
	if e.condition != nil {
		return e.condition(src, g)
	}
	return true
}

func (e *preventDamageRuleContinuous) Apply(g *Game) error {
	toFilter := e.toFactory(e.sourceID)
	g.Effects.AddDamagePreventionRule(WithFrom(e.from), WithTo(toFilter))
	return nil
}

// GrantAttr records a positive delta for the given attr on the given permanent.
// Called by continuous effects during Apply(); the delta is written to
// perm.grantedAttrs at the end of the Apply() cycle.
func (em *EffectManager) GrantAttr(permID uuid.UUID, a Attr) {
	if em.attrDeltas[permID] == nil {
		em.attrDeltas[permID] = make(map[Attr]int)
	}
	em.attrDeltas[permID][a]++
}

// RevokeAttr records a negative delta for the given attr on the given permanent.
// Combined with the permanent's baseAttrs, a net value <= 0 means HasAttr returns false.
func (em *EffectManager) RevokeAttr(permID uuid.UUID, a Attr) {
	if em.attrDeltas[permID] == nil {
		em.attrDeltas[permID] = make(map[Attr]int)
	}
	em.attrDeltas[permID][a]--
}

func (em *EffectManager) Add(e ContinuousEffect) {
	em.effects = append(em.effects, e)
}

func (em *EffectManager) Remove(sourceID uuid.UUID) {
	filtered := em.effects[:0]
	for _, e := range em.effects {
		if e.SourceID() != sourceID {
			filtered = append(filtered, e)
		}
	}
	em.effects = filtered
}

// RemoveEndOfTurn removes all effects with EndOfTurn duration.
func (em *EffectManager) RemoveEndOfTurn() {
	filtered := em.effects[:0]
	for _, e := range em.effects {
		if e.GetDuration() != EndOfTurn {
			filtered = append(filtered, e)
		}
	}
	em.effects = filtered
}

// RemoveEndOfCombat removes all effects with EndOfCombat duration.
func (em *EffectManager) RemoveEndOfCombat() {
	filtered := em.effects[:0]
	for _, e := range em.effects {
		if e.GetDuration() != EndOfCombat {
			filtered = append(filtered, e)
		}
	}
	em.effects = filtered
}

// Apply resets computed bonuses and reapplies all active effects in layer order.
func (em *EffectManager) Apply(g *Game) {
	em.powerBonuses = make(map[uuid.UUID]int)
	em.toughBonuses = make(map[uuid.UUID]int)
	em.attrDeltas = make(map[uuid.UUID]map[Attr]int)
	em.landUntapLimit = -1
	em.unlimitedLandPlays = false
	em.spellCostIncrease = make(map[Color]int)
	em.manaConversion = make(map[Color]Color)
	em.bodyguard = make(map[uuid.UUID]uuid.UUID)
	em.playerDamageRedirect = make(map[uuid.UUID]uuid.UUID)
	// Rebuild prevention rules from continuous effects; preserve one-shot rules (e.g. CoP)
	var oneShotRules []damagePreventionRule
	for _, r := range em.preventionRules {
		if r.oneShot {
			oneShotRules = append(oneShotRules, r)
		}
	}
	em.preventionRules = oneShotRules

	// Reset granted runtime abilities, subtype overrides, and grantedAttrs from effects.
	for _, p := range g.Battlefield {
		if p.FaceDown {
			// Face-down permanents keep their overrides and empty abilities
			continue
		}
		var base []Ability
		for _, a := range p.RuntimeAbilities {
			if _, ok := a.(*grantedByEffect); !ok {
				base = append(base, a)
			}
		}
		p.RuntimeAbilities = base
		p.Controller = p.Card.Owner()
		p.SubTypeOverride = nil
		p.BasePTOverride = nil
		p.ColorOverride = nil
		// Reset grantedAttrs so each Apply() cycle starts fresh.
		p.grantedAttrs = make(map[Attr]int)
	}

	// Remove effects whose source is no longer on the battlefield
	// EndOfTurn and EndOfCombat effects persist until cleanup regardless of source (e.g., Giant Growth, Jade Statue)
	// Indefinite effects persist as long as they self-report active (e.g., Sleight of Mind)
	active := em.effects[:0]
	for _, e := range em.effects {
		dur := e.GetDuration()
		if dur == EndOfTurn || dur == EndOfCombat || dur == Indefinite || g.FindPermanent(e.SourceID()) != nil {
			active = append(active, e)
		}
	}
	em.effects = active

	// Apply in layer order (1, 2, 4, 5, 6, 7)
	for _, layer := range []Layer{LayerCopy, LayerControl, LayerType, LayerColor, LayerAbility, LayerPT} {
		for _, e := range em.effects {
			if e.GetLayer() == layer && e.IsActive(g) {
				_ = e.Apply(g)
			}
		}
	}

	// Sync mana conversions to all player mana pools
	for _, p := range g.Players {
		if len(em.manaConversion) > 0 {
			p.ManaPool().ManaConversions = em.manaConversion
		} else {
			p.ManaPool().ManaConversions = nil
		}
	}

	// Write attrDeltas accumulated by GrantAttr/RevokeAttr calls during this cycle
	// into each permanent's grantedAttrs.
	for permID, deltas := range em.attrDeltas {
		perm := g.FindPermanent(permID)
		if perm == nil {
			continue
		}
		for a, delta := range deltas {
			perm.grantedAttrs[a] += delta
		}
	}
}

func (em *EffectManager) PowerBonus(id uuid.UUID) int {
	return em.powerBonuses[id]
}

func (em *EffectManager) ToughnessBonus(id uuid.UUID) int {
	return em.toughBonuses[id]
}

// SpellCostIncrease returns the additional generic cost for spells of the given color.
func (em *EffectManager) SpellCostIncrease(c Color) int {
	return em.spellCostIncrease[c]
}

// SetPreventCombatDamage marks all combat damage as prevented this turn (Fog, etc.).
func (em *EffectManager) SetPreventCombatDamage() {
	em.preventCombatDamage = true
}

// PreventsCombatDamage returns true if all combat damage is prevented this turn.
func (em *EffectManager) PreventsCombatDamage() bool {
	return em.preventCombatDamage
}

// ClearPreventCombatDamage resets combat damage prevention at end of turn.
func (em *EffectManager) ClearPreventCombatDamage() {
	em.preventCombatDamage = false
}

// grantedByEffect is a marker wrapper to identify abilities granted by continuous effects.
type grantedByEffect struct {
	Ability
}
