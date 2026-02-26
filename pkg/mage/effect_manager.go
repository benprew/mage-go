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
	effects               []ContinuousEffect
	attrDeltas            map[uuid.UUID]map[Attr]int        // deltas accumulated during Apply(); written to perm.grantedAttrs
	blockPairRestrictions map[uuid.UUID]map[uuid.UUID]bool  // attacker -> set of blockers that can't block it; reset each Apply
	Damage                *DamageSystem
	rules                 gameRuleModifiers
}

// gameRuleModifiers groups EffectManager fields related to game rule modifications.
type gameRuleModifiers struct {
	manaConversion     map[Color]Color         // from color -> to color (Sunglasses of Urza)
	spellCostIncrease  map[Color]int           // color -> additional generic cost for spells of that color
	spellCostReduction map[Color]int           // color -> generic cost reduction for spells of that color
	landUntapLimit     int                     // -1 = no limit; >= 0 = max lands that may untap per turn
	artifactUntapLimit int                     // -1 = no limit; >= 0 = max artifacts that may untap per turn
	unlimitedLandPlays bool                    // true if a player can play unlimited lands (Fastbond)
	sanctuaryActive    map[uuid.UUID]bool      // player -> if true, only flying/islandwalk can attack them
	lichActive         map[uuid.UUID]uuid.UUID // player -> source permanent ID of active Lich
	skipNextDraw       map[uuid.UUID]bool      // player -> if true, skip normal draw in draw step
	channelActive      map[uuid.UUID]bool      // players with Channel active this turn
	minimumLife        map[uuid.UUID]bool      // players whose life can't go below 1 (Ali from Cairo)
	maxHandSize        map[uuid.UUID]int       // player -> max hand size override (Cursed Rack)
	expansionCastBlock []string                // expansion names blocked from casting/playing
}

func NewEffectManager() *EffectManager {
	return &EffectManager{
		attrDeltas: make(map[uuid.UUID]map[Attr]int),
		Damage:     NewDamageSystem(),
		rules: gameRuleModifiers{
			landUntapLimit:     -1,
			artifactUntapLimit: -1,
			channelActive:      make(map[uuid.UUID]bool),
			spellCostIncrease:  make(map[Color]int),
			spellCostReduction: make(map[Color]int),
			sanctuaryActive:    make(map[uuid.UUID]bool),
			lichActive:         make(map[uuid.UUID]uuid.UUID),
			skipNextDraw:       make(map[uuid.UUID]bool),
			manaConversion:     make(map[Color]Color),
			minimumLife:        make(map[uuid.UUID]bool),
			maxHandSize:        make(map[uuid.UUID]int),
		},
	}
}

// LandUntapLimit returns the current land untap limit. -1 means no limit.
func (em *EffectManager) LandUntapLimit() int {
	return em.rules.landUntapLimit
}

// HasUnlimitedLandPlays returns true if a player can play unlimited lands this turn.
func (em *EffectManager) HasUnlimitedLandPlays() bool {
	return em.rules.unlimitedLandPlays
}

// SetSanctuaryActive marks a player as protected by Island Sanctuary.
func (em *EffectManager) SetSanctuaryActive(playerID uuid.UUID) {
	em.rules.sanctuaryActive[playerID] = true
}

// IsSanctuaryActive returns true if the player is protected by Island Sanctuary.
func (em *EffectManager) IsSanctuaryActive(playerID uuid.UUID) bool {
	return em.rules.sanctuaryActive[playerID]
}

// ClearSanctuary clears Island Sanctuary protection for a player.
func (em *EffectManager) ClearSanctuary(playerID uuid.UUID) {
	delete(em.rules.sanctuaryActive, playerID)
}

// SetLichActive records the source permanent ID of a Lich controlled by playerID.
func (em *EffectManager) SetLichActive(playerID, sourceID uuid.UUID) {
	em.rules.lichActive[playerID] = sourceID
}

// IsLichActive returns true if the player has an active Lich still on the battlefield.
func (em *EffectManager) IsLichActive(g *Game, playerID uuid.UUID) bool {
	sourceID, ok := em.rules.lichActive[playerID]
	if !ok {
		return false
	}
	return g.FindPermanent(sourceID) != nil
}

// ClearLich clears Lich replacement effects for a player.
func (em *EffectManager) ClearLich(playerID uuid.UUID) {
	delete(em.rules.lichActive, playerID)
}

// SetSkipNextDraw marks a player to skip their next draw step draw.
func (em *EffectManager) SetSkipNextDraw(playerID uuid.UUID) {
	em.rules.skipNextDraw[playerID] = true
}

// ShouldSkipDraw returns true and clears the flag if the player should skip their draw.
func (em *EffectManager) ShouldSkipDraw(playerID uuid.UUID) bool {
	if em.rules.skipNextDraw[playerID] {
		delete(em.rules.skipNextDraw, playerID)
		return true
	}
	return false
}

// SetManaConversion sets a mana color conversion (e.g. Red→White for Sunglasses of Urza).
func (em *EffectManager) SetManaConversion(from, to Color) {
	em.rules.manaConversion[from] = to
}

// GetManaConversion returns the converted color for a given color, if any.
func (em *EffectManager) GetManaConversion(from Color) (Color, bool) {
	to, ok := em.rules.manaConversion[from]
	return to, ok
}

// ClearManaConversion clears all mana conversions.
func (em *EffectManager) ClearManaConversion() {
	em.rules.manaConversion = make(map[Color]Color)
}

// SetChannelActive marks a player as having Channel active this turn.
func (em *EffectManager) SetChannelActive(playerID uuid.UUID) {
	em.rules.channelActive[playerID] = true
}

// IsChannelActive returns true if the player has Channel active.
func (em *EffectManager) IsChannelActive(playerID uuid.UUID) bool {
	return em.rules.channelActive[playerID]
}

// ClearChannelActive clears Channel state (called at end of turn).
func (em *EffectManager) ClearChannelActive() {
	em.rules.channelActive = make(map[uuid.UUID]bool)
}

// SetMinimumLife marks a player as having minimum-life protection (Ali from Cairo).
func (em *EffectManager) SetMinimumLife(playerID uuid.UUID) {
	em.rules.minimumLife[playerID] = true
}

// IsMinimumLifeActive returns true if the player's life can't go below 1.
func (em *EffectManager) IsMinimumLifeActive(playerID uuid.UUID) bool {
	return em.rules.minimumLife[playerID]
}

// ClearMinimumLife resets minimum-life state (called when effect source leaves).
func (em *EffectManager) ClearMinimumLife() {
	em.rules.minimumLife = make(map[uuid.UUID]bool)
}

// AddExpansionCastBlock registers an expansion name as blocked for casting/playing.
func (em *EffectManager) AddExpansionCastBlock(expansion string) {
	em.rules.expansionCastBlock = append(em.rules.expansionCastBlock, expansion)
}

// IsExpansionBlocked returns true if the given expansion is blocked from casting/playing.
func (em *EffectManager) IsExpansionBlocked(expansion string) bool {
	for _, e := range em.rules.expansionCastBlock {
		if e == expansion {
			return true
		}
	}
	return false
}

// ArtifactUntapLimit returns the current artifact untap limit. -1 means no limit.
func (em *EffectManager) ArtifactUntapLimit() int {
	return em.rules.artifactUntapLimit
}

// SetArtifactUntapLimit sets the artifact untap limit.
func (em *EffectManager) SetArtifactUntapLimit(limit int) {
	em.rules.artifactUntapLimit = limit
}

// SetMaxHandSize sets the max hand size override for a player.
func (em *EffectManager) SetMaxHandSize(playerID uuid.UUID, size int) {
	em.rules.maxHandSize[playerID] = size
}

// MaxHandSize returns the max hand size for a player (default 7).
func (em *EffectManager) MaxHandSize(playerID uuid.UUID) int {
	if size, ok := em.rules.maxHandSize[playerID]; ok {
		return size
	}
	return 7
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
	g.Effects.Damage.AddDamagePreventionRule(WithFrom(e.from), WithTo(toFilter))
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

// PreventBlockPair records that the given blocker cannot block the given attacker.
// Reset each Apply cycle; set by continuous effects (e.g. Argothian Pixies).
func (em *EffectManager) PreventBlockPair(blockerID, attackerID uuid.UUID) {
	if em.blockPairRestrictions == nil {
		em.blockPairRestrictions = make(map[uuid.UUID]map[uuid.UUID]bool)
	}
	if em.blockPairRestrictions[attackerID] == nil {
		em.blockPairRestrictions[attackerID] = make(map[uuid.UUID]bool)
	}
	em.blockPairRestrictions[attackerID][blockerID] = true
}

// IsBlockPrevented returns true if the blocker is prevented from blocking the attacker.
func (em *EffectManager) IsBlockPrevented(blockerID, attackerID uuid.UUID) bool {
	if em.blockPairRestrictions == nil {
		return false
	}
	return em.blockPairRestrictions[attackerID][blockerID]
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
	em.attrDeltas = make(map[uuid.UUID]map[Attr]int)
	em.blockPairRestrictions = nil
	em.rules.landUntapLimit = -1
	em.rules.artifactUntapLimit = -1
	em.rules.unlimitedLandPlays = false
	em.rules.maxHandSize = make(map[uuid.UUID]int)
	em.rules.spellCostIncrease = make(map[Color]int)
	em.rules.spellCostReduction = make(map[Color]int)
	em.rules.manaConversion = make(map[Color]Color)
	em.rules.minimumLife = make(map[uuid.UUID]bool)
	em.rules.expansionCastBlock = nil
	em.Damage.ResetPerCycle()

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
		// Reset grantedAttrs and P/T bonuses so each Apply() cycle starts fresh.
		p.grantedAttrs = make(map[Attr]int)
		p.powerBonus = 0
		p.toughBonus = 0
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
		if len(em.rules.manaConversion) > 0 {
			p.ManaPool().ManaConversions = em.rules.manaConversion
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


// SpellCostIncrease returns the additional generic cost for spells of the given color.
func (em *EffectManager) SpellCostIncrease(c Color) int {
	return em.rules.spellCostIncrease[c]
}

// SpellCostReduction returns the generic cost reduction for spells of the given color.
func (em *EffectManager) SpellCostReduction(c Color) int {
	return em.rules.spellCostReduction[c]
}

// grantedByEffect is a marker wrapper to identify abilities granted by continuous effects.
type grantedByEffect struct {
	Ability
}
