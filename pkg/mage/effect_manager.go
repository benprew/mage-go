package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
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

// SourceTapped is active while the source is on the battlefield and tapped.
var SourceTapped ActiveCondition = func(g *Game, sourceID uuid.UUID) bool {
	src := g.FindPermanent(sourceID)
	return src != nil && src.Tapped
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
	// EndOfTurn/EndOfCombat effects are time-limited and don't require their source
	// on the battlefield (e.g. effects from resolved instants/sorceries).
	if e.duration == EndOfTurn || e.duration == EndOfCombat {
		return true
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
	attrDeltas            map[uuid.UUID]map[Attr]int       // deltas accumulated during Apply(); written to perm.grantedAttrs
	blockPairRestrictions map[uuid.UUID]map[uuid.UUID]bool // attacker -> set of blockers that can't block it; reset each Apply
	cantBeBlockedExceptByRules map[uuid.UUID][]PermanentFilter // attacker -> conjunction of filters blockers must match
	canBlockOnlyRules     map[uuid.UUID][]PermanentFilter // blocker -> conjunction of filters attackers must match
	minBlockers           map[uuid.UUID]int               // attacker -> minimum number of blockers required
	replacements          []ReplacementEffect              // persistent: one-shot, turn-scoped, while-on-battlefield
	cycleReplacements     []ReplacementEffect              // cleared each Apply() cycle, re-registered by continuous effects
	Damage                *DamageSystem
	Rules                 *GameRules
}

func NewEffectManager() *EffectManager {
	em := &EffectManager{
		attrDeltas: make(map[uuid.UUID]map[Attr]int),
		Damage:     NewDamageSystem(),
		Rules:      NewGameRules(),
	}
	em.Damage.SetEffectManager(em)
	return em
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
	em.RemoveReplacements(sourceID)
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

// RemoveUntilYourNextTurn removes all effects with UntilYourNextTurn duration
// whose source is controlled by the given player. Called at the beginning of
// each player's upkeep to expire "until your next upkeep" effects.
func (em *EffectManager) RemoveUntilYourNextTurn(g *Game, controllerID uuid.UUID) {
	filtered := em.effects[:0]
	for _, e := range em.effects {
		if e.GetDuration() == UntilYourNextTurn {
			src := g.FindPermanent(e.SourceID())
			if src != nil && src.Controller == controllerID {
				continue // remove this effect
			}
			// Source left battlefield — also remove
			if src == nil {
				continue
			}
		}
		filtered = append(filtered, e)
	}
	em.effects = filtered
}

// Apply resets computed bonuses and reapplies all active effects in layer order.
func (em *EffectManager) Apply(g *Game) {
	em.attrDeltas = make(map[uuid.UUID]map[Attr]int)
	em.blockPairRestrictions = nil
	em.resetCombatRestrictions()
	em.cycleReplacements = nil
	em.Rules.ResetPerCycle()
	em.Damage.ResetPerCycle()

	// Reset granted runtime abilities, subtype overrides, and grantedAttrs from effects.
	for _, p := range g.battlefield {
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
		// Assigning a zero array is a fast memcpy — no allocation.
		p.grantedAttrs = [NumAttrs]int8{}
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
	em.Rules.SyncManaConversions(g.players)

	// Write attrDeltas accumulated by GrantAttr/RevokeAttr calls during this cycle
	// into each permanent's grantedAttrs.
	for permID, deltas := range em.attrDeltas {
		perm := g.FindPermanent(permID)
		if perm == nil {
			continue
		}
		for a, delta := range deltas {
			// delta is ±1 per GrantAttr/RevokeAttr call; realistic stacking
			// from continuous effects is well below the int8 range.
			perm.grantedAttrs[a] += int8(delta)
		}
	}

	// Post-layer enforcement: AttrCantChangeControl reverts any control changes
	// applied during LayerControl. This runs after attrs are written so that
	// effects granted at LayerAbility (e.g. Guardian Beast) take effect.
	for _, p := range g.battlefield {
		if p.HasAttr(AttrCantChangeControl) && p.Controller != p.Card.Owner() {
			p.Controller = p.Card.Owner()
		}
	}
}

// ---------------------------------------------------------------------------
// Replacement effect management
// ---------------------------------------------------------------------------

// AddReplacement adds a persistent replacement effect.
func (em *EffectManager) AddReplacement(r ReplacementEffect) {
	em.replacements = append(em.replacements, r)
}

// PrependReplacement adds a persistent replacement effect at the front of the list
// so it is checked before any existing replacements.
func (em *EffectManager) PrependReplacement(r ReplacementEffect) {
	em.replacements = append([]ReplacementEffect{r}, em.replacements...)
}

// AddCycleReplacement adds a replacement that is cleared each Apply() cycle.
// Used by continuous effects that re-register their replacements each cycle.
func (em *EffectManager) AddCycleReplacement(r ReplacementEffect) {
	em.cycleReplacements = append(em.cycleReplacements, r)
}

// RemoveReplacements removes all replacement effects with the given sourceID.
func (em *EffectManager) RemoveReplacements(sourceID uuid.UUID) {
	filtered := em.replacements[:0]
	for _, r := range em.replacements {
		if r.SourceID() != sourceID {
			filtered = append(filtered, r)
		}
	}
	em.replacements = filtered

	filtered2 := em.cycleReplacements[:0]
	for _, r := range em.cycleReplacements {
		if r.SourceID() != sourceID {
			filtered2 = append(filtered2, r)
		}
	}
	em.cycleReplacements = filtered2
}

// ApplyReplacements runs the replacement pipeline on an action.
// Returns nil if the action was fully prevented/replaced.
// Each replacement fires at most once per event to prevent infinite loops.
func (em *EffectManager) ApplyReplacements(action Action, g *Game) Action {
	applied := make(map[ReplacementEffect]bool)
	tryMatch := func(action Action, preventionOnly bool) (Action, ReplacementEffect, bool) {
		for _, list := range [2][]ReplacementEffect{em.replacements, em.cycleReplacements} {
			for _, r := range list {
				if applied[r] {
					continue
				}
				if isPreventionReplacement(r) != preventionOnly {
					continue
				}
				if !r.IsActive(g) {
					continue
				}
				if r.Matches(action, g) {
					return r.Replace(action, g), r, true
				}
			}
		}
		return action, nil, false
	}
	for {
		if action == nil {
			return nil
		}
		newAction, r, ok := tryMatch(action, false)
		if ok {
			applied[r] = true
			action = newAction
			continue
		}
		newAction, r, ok = tryMatch(action, true)
		if ok {
			applied[r] = true
			action = newAction
			continue
		}
		break
	}
	return action
}

// ClearReplacementsEndOfTurn removes all replacement effects with EndOfTurn duration.
func (em *EffectManager) ClearReplacementsEndOfTurn() {
	filtered := em.replacements[:0]
	for _, r := range em.replacements {
		if r.GetDuration() != EndOfTurn {
			filtered = append(filtered, r)
		}
	}
	em.replacements = filtered
}


// grantedByEffect is a marker wrapper to identify abilities granted by continuous effects.
type grantedByEffect struct {
	Ability
}

// WrapGrantedAbility wraps an ability as granted-by-effect so it is cleaned up
// and re-applied each continuous effect cycle. Use this when a continuous effect
// needs to grant a triggered or activated ability to a permanent.
func WrapGrantedAbility(a Ability) Ability {
	return &grantedByEffect{a}
}
